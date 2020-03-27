package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gbrlsnchs/jwt/v3"
	"github.com/go-sql-driver/mysql"
	"github.com/juleur/ecrpe/graph/generated"
	"github.com/juleur/ecrpe/graph/model"
	"github.com/juleur/ecrpe/interceptors"
	"github.com/juleur/ecrpe/utils"
	"github.com/vektah/gqlparser/gqlerror"
	"golang.org/x/crypto/bcrypt"
)

func (r *mutationResolver) CreateUser(ctx context.Context, input model.NewUserInput) (bool, error) {
	hashPWD, err := bcrypt.GenerateFromPassword([]byte(input.Password), 10)
	if err != nil {
		r.Logger.Errorln(err)
		return false, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	if _, err := r.DB.Exec(
		"INSERT INTO users (username, email, encrypted_pwd, created_at) VALUES (?,?,?,?)",
		input.Username, input.Email, string(hashPWD), time.Now(),
	); err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			r.Logger.Errorln(err)
			if strings.Contains(mysqlErr.Message, "email") {
				return false, &gqlerror.Error{
					Message: "Cette email est déjà utilisée, Veuillez utiliser une autre !",
					Extensions: map[string]interface{}{
						"statusCode": http.StatusConflict,
						"statusText": http.StatusText(http.StatusConflict),
					},
				}
			}
			return false, &gqlerror.Error{
				Message: "Cet username est déjà utilisé, Veuillez en choisir un autre !",
				Extensions: map[string]interface{}{
					"statusCode": http.StatusConflict,
					"statusText": http.StatusText(http.StatusConflict),
				},
			}
		}
		r.Logger.Errorln(err)
		return false, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	return true, nil
}

func (r *mutationResolver) RefreshToken(ctx context.Context, refreshToken string) (*model.Token, error) {
	userAuth := model.UserAuth{}
	if err := r.DB.Get(&userAuth, `
		SELECT ua.user_id, u.username, u.is_teacher FROM user_auths AS ua
		JOIN users AS u ON u.id = ua.user_id
		WHERE ua.is_revoked = 0 AND ua.revoked_at is NULL AND ua.refresh_token = ?
  	`, refreshToken); err != nil {
		r.Logger.Errorln(err)
		e := gqlerror.Error{
			Message: "Oops, une erreur est survenue avec votre session, veuillez vous réauthentifier",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusUnauthorized,
				"statusText": http.StatusText(http.StatusUnauthorized),
			},
		}
		fmt.Println(e.Error())
		return &model.Token{}, &e
	}
	// revoke token
	if _, err := r.DB.Queryx(`
	    UPDATE user_auths SET is_revoked=?, revoked_at=?
	    WHERE is_revoked=0 AND revoked_at is NULL AND user_id=? AND refresh_token=?
	`, 1, time.Now(), userAuth.UserID, refreshToken); err != nil {
		r.Logger.Errorln(err)
		return &model.Token{}, &gqlerror.Error{
			Message: "Oops, une erreur est survenue avec votre session, veuillez vous réauthentifier",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusUnauthorized,
				"statusText": http.StatusText(http.StatusUnauthorized),
			},
		}
	}
	// Create new jwt then new refresh token
	pl := model.CustomPayload{
		Payload: jwt.Payload{
			Issuer:         "https://rf.ecrpe.fr",
			ExpirationTime: jwt.NumericDate(time.Now().Add(1 * time.Minute)),
			IssuedAt:       jwt.NumericDate(time.Now()),
		},
		Username: userAuth.Username,
		UserID:   userAuth.UserID,
		Teacher:  userAuth.IsTeacher,
	}
	jwtoken, err := jwt.Sign(pl, jwt.NewHS512([]byte(r.SecretKey)))
	if err != nil {
		r.Logger.Errorln(err)
		return &model.Token{}, &gqlerror.Error{
			Message: "Oops, une erreur est survenue avec votre session, veuillez vous réauthentifier",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusUnauthorized,
				"statusText": http.StatusText(http.StatusUnauthorized),
			},
		}
	}
	tokens := model.Token{
		Jwt:          string(jwtoken),
		RefreshToken: utils.HexKeyGenerator(16),
	}

	// Get IP Address from user
	userIP := interceptors.ForIPAddress(ctx)
	userAgent := interceptors.ForUserAgent(ctx)
	// push refresh token
	if _, err = r.DB.Exec(`
	    INSERT INTO user_auths (user_agent, ip_address, refresh_token, delivered_at, on_refresh, user_id)
	    VALUES (?,?,?,?,?,?)
	  `, userAgent, userIP, tokens.RefreshToken, time.Now(), 1, userAuth.UserID,
	); err != nil {
		r.Logger.Errorln(err)
		return &model.Token{}, &gqlerror.Error{
			Message: "Oops, une erreur est survenue avec votre session, veuillez vous réauthentifier",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusUnauthorized,
				"statusText": http.StatusText(http.StatusUnauthorized),
			},
		}
	}
	return &tokens, nil
}

func (r *mutationResolver) UpdateUser(ctx context.Context, input model.UpdateUserInput) (*model.User, error) {
	if *input.Email == "" && *input.Username == "" {
		return &model.User{}, &gqlerror.Error{
			Message: "Rien à mettre à jour",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusContinue,
				"statusText": http.StatusText(http.StatusContinue),
			},
		}
	}
	userAuth := interceptors.ForUserContext(ctx)
	if !userAuth.IsAuth {
		r.Logger.Errorln(fmt.Sprintf("User n°%d authentication didn't succeed", userAuth.UserID), "HttpErrorStatus", userAuth.HttpErrorResponse.StatusText)
		return &model.User{}, &gqlerror.Error{
			Message: userAuth.HttpErrorResponse.Message,
			Extensions: map[string]interface{}{
				"statusCode": userAuth.HttpErrorResponse.StatusCode,
				"statusText": userAuth.HttpErrorResponse.StatusText,
			},
		}
	}
	user := model.User{}
	// fetch user password before bcrypt checking
	if err := r.DB.Get(&user, "SELECT encrypted_pwd FROM users WHERE id = ?", userAuth.UserID); err != nil {
		r.Logger.Errorln(err)
		return &model.User{}, &gqlerror.Error{
			Message: "Oops, nous n'avons pu procéder à la mise à jour de votre profil, veuillez contacter l'administrateur !",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	// checking if password is correct
	if err := bcrypt.CompareHashAndPassword([]byte(user.EncryptedPWD), []byte(input.Password)); err != nil {
		r.Logger.Errorln(err)
		return &model.User{}, &gqlerror.Error{
			Message: "Votre mot de passe est incorrect, nous n'avons pu procéder à la mise à jour de votre profil",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusForbidden,
				"statusText": http.StatusText(http.StatusForbidden),
			},
		}
	}

	query := strings.Builder{}
	query.WriteString("UPDATE users SET ")
	argsQuery := []string{}
	userUpdated := model.User{}
	if *input.Email != "" {
		argsQuery = append(argsQuery, fmt.Sprintf("email = '%s'", *input.Email))
		userUpdated.Email = *input.Email
	}
	if *input.Username != "" {
		argsQuery = append(argsQuery, fmt.Sprintf("username = '%s'", *input.Username))
		userUpdated.Username = *input.Username
	}
	query.WriteString(strings.Join(argsQuery, ","))
	query.WriteString(fmt.Sprintf(" WHERE id = '%d'", userAuth.UserID))

	if _, err := r.DB.Queryx(query.String()); err != nil {
		r.Logger.Errorln(err)
		return &model.User{}, &gqlerror.Error{
			Message: "Oops, nous n'avons pu procéder à la mise à jour de votre profil, veuillez contacter l'administrateur",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	return &userUpdated, nil
}

func (r *mutationResolver) PurchaseRefresherCourse(ctx context.Context, input model.PurchaseRefresherCourseInput) (bool, error) {
	userAuth := interceptors.ForUserContext(ctx)
	if !userAuth.IsAuth {
		r.Logger.Errorln(fmt.Sprintf("User n°%d authentication didn't succeed", userAuth.UserID), "HttpErrorStatus", userAuth.HttpErrorResponse.StatusText)
		return false, &gqlerror.Error{
			Message: userAuth.HttpErrorResponse.Message,
			Extensions: map[string]interface{}{
				"statusCode": userAuth.HttpErrorResponse.StatusCode,
				"statusText": userAuth.HttpErrorResponse.StatusText,
			},
		}
	}
	// create payments before
	paymentsIDRes, err := r.DB.Exec(
		"INSERT INTO payments (paypal_payer_id, paypal_order_id, created_at) VALUES (?,?,?)",
		input.PaypalPayerID, input.PaypalOrderID, time.Now())
	if err != nil {
		r.Logger.Errorln(err)
		return false, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, veuillez réessayer puis contacter l'administrateur en cas de nouvelle erreur",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	paymentsID, err := paymentsIDRes.LastInsertId()
	if err != nil {
		r.Logger.Errorln(err)
		return false, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, veuillez réessayer puis contacter l'administrateur en cas de nouvelle erreur",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	//
	if _, err := r.DB.Exec(
		"INSERT INTO users_refresher_courses (payment_id, user_id, refresher_course_id) VALUES (?,?,?)",
		paymentsID, userAuth.UserID, input.RefresherCourseID,
	); err != nil {
		r.Logger.Errorln(err)
		return false, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, veuillez réessayer puis contacter l'administrateur en cas de nouvelle erreur",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	return true, nil
}

func (r *mutationResolver) CreateRefresherCourse(ctx context.Context, input model.NewSessionInput) (bool, error) {
	// check if teacher
	userAuth := interceptors.ForUserContext(ctx)
	if !userAuth.IsAuth {
		r.Logger.Errorln(fmt.Sprintf("User n°%d authentication didn't succeed", userAuth.UserID), "HttpErrorStatus", userAuth.HttpErrorResponse.StatusText)
		return false, &gqlerror.Error{
			Message: userAuth.HttpErrorResponse.Message,
			Extensions: map[string]interface{}{
				"statusCode": userAuth.HttpErrorResponse.StatusCode,
				"statusText": userAuth.HttpErrorResponse.StatusText,
			},
		}
	}

	if _, err := r.DB.Queryx("SELECT id FROM users WHERE id = ? AND is_teacher = 1", userAuth.UserID); err != nil {
		r.Logger.Errorln(err)
		return false, &gqlerror.Error{
			Extensions: map[string]interface{}{
				"statusCode": http.StatusForbidden,
				"statusText": http.StatusText(http.StatusForbidden),
			},
		}
	}
	refCourse := model.RefresherCourse{}
	if err := r.DB.Get(&refCourse, `
    	SELECT id, subject, year FROM refresher_courses WHERE id = ?
  	`, input.RefresherCourseID); err != nil {
		r.Logger.Errorln(err)
		return false, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, veuillez réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	// Create new session
	sessionIDRes, err := r.DB.Exec(`
    	INSERT INTO sessions (title, section, type, description, session_number, recorded_on, created_at, refresher_course_id, user_id) VALUES (?,?,?,?,?,?,?,?,?)
  	`, input.Title, input.Section, input.Type, input.Description, input.SessionNumber, input.RecordedOn, time.Now(), input.RefresherCourseID, userAuth.UserID)
	if err != nil {
		r.Logger.Errorln(err)
		return false, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, veuillez réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	sessionID, err := sessionIDRes.LastInsertId()
	if err != nil {
		r.Logger.Errorln(err)
		return false, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, veuillez réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	// directory path
	dirPath := fmt.Sprintf("/player/%s/%s/rc/session-%d", strings.ToLower(refCourse.Subject.String()), *refCourse.Year, sessionID)

	go r.UploadFileManager.ProcessVideo(dirPath, int(sessionID), input.VideoFile, refCourse)
	if len(input.DocFiles) > 0 {
		go r.UploadFileManager.ProcessDoc(dirPath, int(sessionID), input.DocFiles)
	}

	return true, nil
}

func (r *queryResolver) Login(ctx context.Context, input model.LoginInput) (*model.Token, error) {
	user := model.User{}
	if err := r.DB.Get(&user, "SELECT id, username, is_teacher, encrypted_pwd FROM users WHERE email = ?", input.Email); err != nil {
		if err == sql.ErrNoRows {
			r.Logger.Errorln(err)
			return &model.Token{}, &gqlerror.Error{
				Message: "L'email et le Mot de Passe saisis ne correspondent pas à de nos archives, veuillez vérifier vos identifiants puis réessayez",
				Extensions: map[string]interface{}{
					"statusCode": http.StatusNotFound,
					"statusText": http.StatusText(http.StatusNotFound),
				},
			}
		}
		r.Logger.Errorln(err)
		return &model.Token{}, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	// check if password matches with the one in db
	if err := bcrypt.CompareHashAndPassword([]byte(user.EncryptedPWD), []byte(input.Password)); err != nil {
		r.Logger.Errorln(err)
		return &model.Token{}, &gqlerror.Error{
			Message: "L'email et le Mot de Passe saisis ne correspondent à aucunes de nos archives, veuillez vérifier vos identifiants puis réessayez !",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusNotFound,
				"statusText": http.StatusText(http.StatusNotFound),
			},
		}
	}
	// revokes last refresh token from user
	if _, err := r.DB.Queryx(`
		UPDATE user_auths SET is_revoked=?, revoked_at=?
		WHERE is_revoked = 0 AND revoked_at is NULL AND user_id = ?
		ORDER BY delivered_at DESC
		LIMIT 1
  	`, 1, time.Now(), user.ID); err != nil {
		r.Logger.Errorln(err)
		return &model.Token{}, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}

	// generate new tokens
	pl := model.CustomPayload{
		Payload: jwt.Payload{
			Issuer:         "https://rf.ecrpe.fr",
			ExpirationTime: jwt.NumericDate(time.Now().Add(1 * time.Minute)), // 12 hours
			IssuedAt:       jwt.NumericDate(time.Now()),
		},
		Username: user.Username,
		UserID:   user.ID,
		Teacher:  user.IsTeacher,
	}
	jwtoken, err := jwt.Sign(pl, jwt.NewHS512([]byte(r.SecretKey)))
	if err != nil {
		r.Logger.Errorln(err)
		return &model.Token{}, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	tokens := model.Token{
		Jwt:          string(jwtoken),
		RefreshToken: utils.HexKeyGenerator(16),
	}
	// push tokens
	userIP := interceptors.ForIPAddress(ctx)
	userAgent := interceptors.ForUserAgent(ctx)
	if _, err = r.DB.Exec(`
    INSERT INTO user_auths (user_agent, ip_address, refresh_token, delivered_at, on_login, user_id) VALUES (?,?,?,?,?,?)
  `, userAgent, userIP, tokens.RefreshToken, time.Now(), 1, user.ID); err != nil {
		r.Logger.Errorln(err)
		return &model.Token{}, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	return &tokens, nil
}

func (r *queryResolver) RefresherCourses(ctx context.Context, input model.RefresherCourseInput) ([]*model.RefresherCourse, error) {
	rc := make([]*model.RefresherCourse, 0)
	if input.ByUserID == nil && input.BySubject == nil {
		if err := r.DB.Select(&rc, "SELECT * FROM refresher_courses"); err != nil {
			r.Logger.Errorln(err)
			return rc, &gqlerror.Error{
				Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
				Extensions: map[string]interface{}{
					"statusCode": http.StatusInternalServerError,
					"statusText": http.StatusText(http.StatusInternalServerError),
				},
			}
		}
		return rc, nil
	}
	if input.ByUserID != nil {
		if err := r.DB.Select(&rc, `
			SELECT rc.* FROM refresher_courses AS rc
			JOIN users_refresher_courses AS urc ON rc.id = urc.refresher_course_id
			WHERE urc.user_id = ?
		`, input.ByUserID); err != nil {
			r.Logger.Errorln(err)
			return rc, &gqlerror.Error{
				Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
				Extensions: map[string]interface{}{
					"statusCode": http.StatusInternalServerError,
					"statusText": http.StatusText(http.StatusInternalServerError),
				},
			}
		}
	} else {
		if err := r.DB.Select(rc, "SELECT * FROM refresher_courses WHERE subject = ?", input.BySubject); err != nil {
			r.Logger.Errorln(err)
			return rc, &gqlerror.Error{
				Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
				Extensions: map[string]interface{}{
					"statusCode": http.StatusInternalServerError,
					"statusText": http.StatusText(http.StatusInternalServerError),
				},
			}
		}
	}
	return rc, nil
}

func (r *queryResolver) RefresherCourse(ctx context.Context, refresherCourseID int) (*model.RefresherCourseResponse, error) {
	refCourse := model.RefresherCourse{}
	if err := r.DB.Get(&refCourse, "SELECT * FROM refresher_courses WHERE id = ?", refresherCourseID); err != nil {
		if err == sql.ErrNoRows {
			r.Logger.Errorln(err)
			return &model.RefresherCourseResponse{}, &gqlerror.Error{
				Message: "Désolé, nous ne pouvons trouver ce cours",
				Extensions: map[string]interface{}{
					"statusCode": http.StatusNotFound,
					"statusText": http.StatusText(http.StatusNotFound),
				},
			}
		}
		r.Logger.Errorln(err)
		return &model.RefresherCourseResponse{}, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	sessions := make([]*model.Session, 0)
	if err := r.DB.Select(&sessions, `
		SELECT id, title, section, type, description, session_number, recorded_on, created_at, updated_at FROM sessions WHERE refresher_course_id = ? AND is_ready = 1
	`, refresherCourseID); err != nil {
		r.Logger.Errorln(err)
		return &model.RefresherCourseResponse{}, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	return &model.RefresherCourseResponse{RefresherCourse: &refCourse, Sessions: sessions}, nil
}

func (r *queryResolver) PlayerCheckUser(ctx context.Context) (bool, error) {
	userAuth := interceptors.ForUserContext(ctx)
	if !userAuth.IsAuth {
		r.Logger.Errorln(fmt.Sprintf("User n°%d authentication didn't succeed", userAuth.UserID), "HttpErrorStatus", userAuth.HttpErrorResponse.StatusText)
		return false, &gqlerror.Error{
			Message: userAuth.HttpErrorResponse.Message,
			Extensions: map[string]interface{}{
				"statusCode": userAuth.HttpErrorResponse.StatusCode,
				"statusText": userAuth.HttpErrorResponse.StatusText,
			},
		}
	}
	userIPAddress := interceptors.ForIPAddress(ctx)
	lastIPCached, ok := r.RedisCache.GetIP(string(userAuth.UserID))
	if !ok {
		return false, &gqlerror.Error{
			Message: "Oops; une erreur est survenue, veuillez vous réauthentifier",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusForbidden,
				"statusText": http.StatusText(http.StatusForbidden),
			},
		}
	}
	err := utils.IPsChecker(userIPAddress, lastIPCached)
	if err != nil {
		r.Logger.Errorln(err)
		r.RedisCache.DeleteIP(string(userAuth.UserID))
		return false, &gqlerror.Error{
			Message: err.Error(),
			Extensions: map[string]interface{}{
				"statusCode": http.StatusForbidden,
				"statusText": http.StatusText(http.StatusForbidden),
			},
		}
	}
	// overwrite default ttl from 24 hours to 10 minutes for checking IPs while user watching course
	r.RedisCache.AddIP(string(userAuth.UserID), userIPAddress, 15*time.Minute)
	return true, nil
}

func (r *queryResolver) Profile(ctx context.Context, userID int) (*model.User, error) {
	userAuth := interceptors.ForUserContext(ctx)
	if !userAuth.IsAuth {
		r.Logger.Errorln(fmt.Sprintf("User n°%d authentication didn't succeed", userAuth.UserID), "HttpErrorStatus", userAuth.HttpErrorResponse.StatusText)
		return &model.User{}, &gqlerror.Error{
			Message: userAuth.HttpErrorResponse.Message,
			Extensions: map[string]interface{}{
				"statusCode": userAuth.HttpErrorResponse.StatusCode,
				"statusText": userAuth.HttpErrorResponse.StatusText,
			},
		}
	}
	user := model.User{}
	if err := r.DB.Get(&user, `
		SELECT id, username, email, created_at, updated_at FROM users WHERE id = ?
	`, userID); err != nil {
		r.Logger.Errorln(err)
		return &user, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, veuillez vous réauthentifier",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	return &user, nil
}

func (r *queryResolver) SessionCourse(ctx context.Context, input model.SessionInput) (*model.SessionResponse, error) {
	if _, err := r.DB.Queryx(`
		SELECT user_id FROM users_refresher_courses
		WHERE user_id = ? AND refresher_course_id = ?
	`, input.UserID, input.RefresherCourseID); err != nil {
		if err == sql.ErrNoRows {
			r.Logger.Errorln(err)
			return &model.SessionResponse{}, &gqlerror.Error{
				Message: "Vous n'avez pas acheté ce cours",
				Extensions: map[string]interface{}{
					"statusCode": http.StatusForbidden,
					"statusText": http.StatusText(http.StatusForbidden),
				},
			}
		}
		r.Logger.Errorln(err)
		return &model.SessionResponse{}, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	session := model.Session{}
	if err := r.DB.Get(&session, `
		SELECT id, title, section, type, description, session_number, recorded_on, created_at, updated_at FROM sessions
		WHERE id = ? AND refresher_course_id = ? AND is_ready = 1
	`, input.SessionID, input.RefresherCourseID); err != nil {
		r.Logger.Errorln(err)
		return &model.SessionResponse{}, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	video := model.Video{}
	if err := r.DB.Get(&video, `
		SELECT id, path, created_at, updated_at FROM videos WHERE session_id = ?
	`, input.SessionID); err != nil {
		r.Logger.Errorln(err)
		return &model.SessionResponse{}, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	video.Path = video.Path[20:]
	classPapers := make([]*model.ClassPaper, 0)
	if err := r.DB.Select(&classPapers, `
		SELECT id, title, path, created_at, updated_at FROM class_papers WHERE session_id = ?
	`, input.SessionID); err != nil {
		r.Logger.Errorln(err)
		return &model.SessionResponse{}, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	classPapers = utils.ClassPapersPathRewrite(classPapers)
	teacher := model.User{}
	if err := r.DB.Get(&teacher, `
		SELECT u.id, u.username, u.fullname FROM users AS u
		JOIN sessions AS s ON s.user_id = u.id
		WHERE u.id = ? AND u.is_teacher = 1
	`, input.UserID); err != nil {
		r.Logger.Errorln(err)
		return &model.SessionResponse{}, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	return &model.SessionResponse{Session: &session, Video: &video, ClassPapers: classPapers, Teacher: &teacher}, nil
}

func (r *queryResolver) AuthTeacher(ctx context.Context, userID int) (bool, error) {
	user := model.User{}
	if err := r.DB.Get(&user, "SELECT is_teacher FROM users WHERE id = ?", userID); err != nil {
		if err == sql.ErrNoRows {
			r.Logger.Errorln(err)
			return false, &gqlerror.Error{
				Message: "Vous n'avez pas accès au portail des professeurs",
				Extensions: map[string]interface{}{
					"statusCode": http.StatusForbidden,
					"statusText": http.StatusText(http.StatusForbidden),
				},
			}
		}
		r.Logger.Errorln(err)
		return false, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	if user.IsTeacher {
		return true, nil
	}
	return false, &gqlerror.Error{
		Message: "Vous n'avez pas accès au portail des professeurs",
		Extensions: map[string]interface{}{
			"statusCode": http.StatusForbidden,
			"statusText": http.StatusText(http.StatusForbidden),
		},
	}
}

func (r *queryResolver) SubjectsEnum(ctx context.Context) ([]string, error) {
	// maybe better
	subs := []string{"ECONOMICS", "FRENCH", "MATHETIMATICS"}
	return subs, nil
}

func (r *queryResolver) TotalHoursCourses(ctx context.Context) (string, error) {
	durations := []string{}
	if err := r.DB.Select(&durations, "SELECT duration FROM videos"); err != nil {
		r.Logger.Errorln(err)
		return "", &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	return utils.DurationCounter(durations), nil
}

func (r *refresherCourseResolver) TotalDuration(ctx context.Context, obj *model.RefresherCourse) (*string, error) {
	var totalDuration []string
	var ttDur string
	if err := r.DB.Select(&totalDuration, `
		SELECT duration FROM videos AS v JOIN sessions AS s ON v.session_id = s.id
		WHERE s.refresher_course_id = ?
	`, obj.ID); err != nil {
		r.Logger.Errorln(err)
		return &ttDur, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	ttDur = utils.DurationCounter(totalDuration)
	return &ttDur, nil
}

func (r *refresherCourseResolver) IsPurchased(ctx context.Context, obj *model.RefresherCourse) (*bool, error) {
	f := false
	if user := interceptors.ForUserContext(ctx); user.IsAuth {
		var pId int
		if err := r.DB.Get(&pId, `
      SELECT payment_id FROM users_refresher_courses AS urc
      WHERE urc.refresher_course_id = ? AND urc.user_id = ?
    `, obj.ID, user.UserID); err != nil {
			if pId == 0 {
				return &f, nil
			}
			r.Logger.Errorln(err)
			return &f, &gqlerror.Error{
				Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
				Extensions: map[string]interface{}{
					"statusCode": http.StatusInternalServerError,
					"statusText": http.StatusText(http.StatusInternalServerError),
				},
			}
		}
		t := true
		return &t, nil
	}
	return &f, nil
}

func (r *refresherCourseResolver) Teachers(ctx context.Context, obj *model.RefresherCourse) ([]*model.User, error) {
	teachers := make([]*model.User, 0)
	if err := r.DB.Select(&teachers, `
		SELECT id, username FROM users WHERE id IN(
			SELECT DISTINCT user_id FROM sessions WHERE refresher_course_id = ?
		)
	`, obj.ID); err != nil {
		r.Logger.Errorln(err)
		return teachers, &gqlerror.Error{
			Message: "Oops, une erreur est survenue, merci de réessayer ultérieurement",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	}
	return teachers, nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// RefresherCourse returns generated.RefresherCourseResolver implementation.
func (r *Resolver) RefresherCourse() generated.RefresherCourseResolver {
	return &refresherCourseResolver{r}
}

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type refresherCourseResolver struct{ *Resolver }
