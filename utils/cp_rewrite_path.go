package utils

import "github.com/juleur/becrpe/graph/model"

func ClassPapersPathRewrite(cps []*model.ClassPaper) []*model.ClassPaper {
	for _, cp := range cps {
		cp.Path = cp.Path[20:]
	}
	return cps
}
