package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/99designs/gqlgen/graphql"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type UploadFileManager struct {
	VideoCh      chan Video
	ClassPaperCh chan ClassPaper
	Logger       *logrus.Logger
	DB           *sqlx.DB
}

func NewUploadFileManager(db *sqlx.DB, logger *logrus.Logger) *UploadFileManager {
	ufm := &UploadFileManager{
		VideoCh:      make(chan Video, 50),
		ClassPaperCh: make(chan ClassPaper, 50),
		DB:           db,
		Logger:       logger,
	}
	return ufm
}

func (ufm *UploadFileManager) DoneProcesses() {
	for {
		select {
		case videoDone := <-ufm.VideoCh:
			if _, err := ufm.DB.Exec(
				"INSERT INTO videos (path, duration, created_at, session_id) VALUES (?,?,?,?)",
				videoDone.Path, videoDone.Duration, time.Now(), videoDone.SessionID,
			); err != nil {
				ufm.Logger.Errorln(err)
			}
			if _, err := ufm.DB.Queryx(`
				UPDATE sessions SET is_ready = 1, updated_at = ? WHERE id = ?
			`, time.Now(), videoDone.SessionID); err != nil {
				ufm.Logger.Errorln(err)
			}
		case cpDone := <-ufm.ClassPaperCh:
			if _, err := ufm.DB.Exec(
				"INSERT INTO class_papers (title, path, created_at, session_id) VALUES (?,?,?,?)",
				cpDone.Title, cpDone.Path, time.Now(), cpDone.SessionID,
			); err != nil {
				ufm.Logger.Errorln(err)
			}
		}
	}
}

func (ufm *UploadFileManager) ProcessVideo(dirPath string, sessionID int, videoFile graphql.Upload, refCourse RefresherCourse) {
	// video.xxxxxxx
	videoTmpFile, err := ioutil.TempFile(os.TempDir(), "video.*")
	if err != nil {
		ufm.Logger.Errorln(err)
		return
	}
	defer os.Remove(videoTmpFile.Name())

	fileBytes := bytes.Buffer{}
	fileBytes.Grow(int(videoFile.Size))
	if _, err = fileBytes.ReadFrom(videoFile.File); err != nil {
		ufm.Logger.Errorln(err)
		return
	}
	if _, err := videoTmpFile.Write(fileBytes.Bytes()); err != nil {
		ufm.Logger.Errorln(err)
		return
	}
	// split /tmp/video.xxxxxxx
	videoName := strings.Split(videoTmpFile.Name(), "/")
	fragMP4 := fmt.Sprintf("mp4fragment --fragment-duration 4000 %s %s-f.mp4", videoTmpFile.Name(), videoTmpFile.Name())
	cmd := exec.Command("bash", "-c", fragMP4)
	if err := cmd.Run(); err != nil {
		ufm.Logger.Error(err)
		return
	}
	defer os.Remove(videoTmpFile.Name() + "-f.mp4")

	dashMP4 := fmt.Sprintf("mp4dash --language-map=en:fr,und:fr --media-prefix %s --mpd-name %s.mpd --profiles on-demand --use-segment-timeline %s-f.mp4 -f -o %s", videoName[len(videoName)-1], videoName[len(videoName)-1], videoTmpFile.Name(), os.TempDir())
	cmd = exec.Command("bash", "-c", dashMP4)
	if err := cmd.Run(); err != nil {
		ufm.Logger.Errorln(err)
		return
	}
	defer os.Remove(videoTmpFile.Name() + "-video-avc1.mp4")
	defer os.Remove(videoTmpFile.Name() + "-audio-fr-mp4a.mp4")
	defer os.Remove(videoTmpFile.Name() + ".mpd")

	argsFfprobe := []string{
		"-show_entries", "format=duration", "-v", "quiet", "-of", "csv=p=0",
		"-sexagesimal", videoTmpFile.Name(),
	}
	durVideo, err := exec.Command("ffprobe", argsFfprobe...).Output()
	if err != nil {
		ufm.Logger.Errorln(err)
		return
	}
	videoFluxFile, err := os.OpenFile(videoTmpFile.Name()+"-video-avc1.mp4", os.O_RDONLY, 0444)
	if err != nil {
		ufm.Logger.Errorln(err)
		return
	}
	defer videoFluxFile.Close()
	audioFluxFile, err := os.OpenFile(videoTmpFile.Name()+"-audio-fr-mp4a.mp4", os.O_RDONLY, 0444)
	if err != nil {
		ufm.Logger.Errorln(err)
		return
	}
	defer audioFluxFile.Close()
	mpdFile, err := os.OpenFile(videoTmpFile.Name()+".mpd", os.O_RDONLY, 0444)
	if err != nil {
		ufm.Logger.Errorln(err)
		return
	}
	defer mpdFile.Close()

	// http request to o2switch
	finalDirPath, err := ufm.sendVideoFiles(dirPath, videoFluxFile, audioFluxFile, mpdFile)
	if err != nil {
		ufm.Logger.Errorln(err)
		return
	}

	video := Video{
		Path:      finalDirPath,
		Duration:  prettifyDurationOutput(durVideo),
		SessionID: sessionID,
	}
	ufm.VideoCh <- video
}

func (ufm *UploadFileManager) ProcessDoc(dirPath string, sessionID int, docUploadFiles []*DocUploadFile) {
	for _, docUploadFile := range docUploadFiles {
		var fileName string
		splitFile := strings.SplitAfter(docUploadFile.File.Filename, ".")
		if docUploadFile.Title == nil {
			fileName = splitFile[0]
		} else {
			fileName = strings.Replace(normSubject(*docUploadFile.Title), " ", "_", -1)
		}
		tmpFile, err := ioutil.TempFile(os.TempDir(), fileName+".*."+splitFile[1])
		if err != nil {
			ufm.Logger.Errorln(err)
			return
		}
		defer os.Remove(tmpFile.Name())
		// 20mb maxi
		if docUploadFile.File.Size > 20000000 {
			ufm.Logger.Errorln(fmt.Errorf("%s size is too big", fileName))
			return
		}
		fileBytes := bytes.Buffer{}
		fileBytes.Grow(int(docUploadFile.File.Size))
		if _, err := fileBytes.ReadFrom(docUploadFile.File.File); err != nil {
			ufm.Logger.Errorln(err)
			return
		}

		docFile, err := os.OpenFile(tmpFile.Name(), os.O_RDONLY, 0444)
		if err != nil {
			ufm.Logger.Errorln(err)
			return
		}
		defer docFile.Close()

		// http request to o2switch
		finalDirPath, err := ufm.sendDocumentFile(dirPath, docFile)
		if err != nil {
			ufm.Logger.Errorln(err)
			return
		}

		ufm.ClassPaperCh <- ClassPaper{
			Title:     fileName,
			Path:      finalDirPath,
			SessionID: sessionID,
		}
	}
}

func (ufm *UploadFileManager) sendVideoFiles(dirPath string, vFile, aFile, mpdFile *os.File) (string, error) {
	const urlServer = "http://localhost:8080/storage_video"
	r, w := io.Pipe()
	m := multipart.NewWriter(w)

	go func() {
		defer w.Close()
		defer m.Close()
		if err := m.WriteField("dir_path", dirPath); err != nil {
			ufm.Logger.Errorln(err)
			return
		}
		// video
		filename := strings.Split(vFile.Name(), "/")
		part, err := m.CreateFormFile("vfile", filename[len(filename)-1])
		if err != nil {
			ufm.Logger.Errorln(err)
			return
		}
		file, err := os.Open(vFile.Name())
		if err != nil {
			ufm.Logger.Errorln(err)
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			ufm.Logger.Errorln(err)
			return
		}
		// audio
		filename = strings.Split(aFile.Name(), "/")
		part, err = m.CreateFormFile("afile", filename[len(filename)-1])
		if err != nil {
			ufm.Logger.Errorln(err)
			return
		}
		file, err = os.Open(aFile.Name())
		if err != nil {
			ufm.Logger.Errorln(err)
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			ufm.Logger.Errorln(err)
			return
		}
		// mpd
		filename = strings.Split(mpdFile.Name(), "/")
		part, err = m.CreateFormFile("mpdfile", filename[len(filename)-1])
		if err != nil {
			ufm.Logger.Errorln(err)
			return
		}
		file, err = os.Open(mpdFile.Name())
		if err != nil {
			ufm.Logger.Errorln(err)
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			ufm.Logger.Errorln(err)
			return
		}
		defer file.Close()
	}()
	resp, err := http.Post(urlServer, m.FormDataContentType(), r)
	if err != nil {
		ufm.Logger.Errorln(err)
		return "", err
	}
	rBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ufm.Logger.Errorln(err)
		return "", err
	}
	var dat map[string]interface{}
	if err := json.Unmarshal(rBody, &dat); err != nil {
		ufm.Logger.Errorln(err)
		return "", err
	}
	dirP := dat["dir_path"].(string)
	return dirP, nil
}

func (ufm *UploadFileManager) sendDocumentFile(dirPath string, doc *os.File) (string, error) {
	const urlServer = "http://localhost:8080/storage_doc"
	r, w := io.Pipe()
	m := multipart.NewWriter(w)

	go func() {
		w.Close()
		m.Close()
		if err := m.WriteField("dir_path", dirPath); err != nil {
			ufm.Logger.Errorln(err)
			return
		}
		// doc
		part, err := m.CreateFormFile("docfile", doc.Name())
		if err != nil {
			ufm.Logger.Errorln(err)
			return
		}
		file, err := os.Open(doc.Name())
		if err != nil {
			ufm.Logger.Errorln(err)
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			ufm.Logger.Errorln(err)
			return
		}
	}()
	// doc field
	resp, err := http.Post(urlServer, m.FormDataContentType(), r)
	if err != nil {
		ufm.Logger.Errorln(err)
		return "", err
	}
	rBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ufm.Logger.Errorln(err)
		return "", err
	}
	var dat map[string]interface{}
	if err := json.Unmarshal(rBody, &dat); err != nil {
		ufm.Logger.Errorln(err)
		return "", err
	}
	dirP := dat["dir_path"].(string)
	return dirP, nil
}

func prettifyDurationOutput(durationInMS []byte) string {
	// remove \n from cmd output
	outputWithoutNewline := strings.TrimSuffix(string(durationInMS), "\n")
	// keep 3:23:54 format
	return strings.Split(outputWithoutNewline, ".")[0]
}

func normSubject(subjectName string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	sLower := strings.ToLower(subjectName)
	r, _, err := transform.String(t, sLower)
	if err != nil {
		return sLower
	}
	return r
}
