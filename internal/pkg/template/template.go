package template

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/ic3network/mccs-alpha-api/global/constant"
	"github.com/ic3network/mccs-alpha-api/internal/pkg/e"
	"github.com/ic3network/mccs-alpha-api/internal/pkg/flash"
)

var (
	layoutDir   = "web/template/layout/"
	templateExt = ".html"
)

type View struct {
	Template *template.Template
	Layout   string
}

func NewView(templateName string) *View {
	templates := append(layoutFiles(), "web/template/"+templateName+".html")

	t, err := template.New("").
		Funcs(template.FuncMap{
			"ArrToSting":           arrToSting,
			"TagsToString":         tagsToString,
			"TagsToSearchString":   tagsToSearchString,
			"Add":                  add,
			"Minus":                minus,
			"N":                    n,
			"IDToString":           idToString,
			"FormatTime":           formatTime,
			"FormatAccountBalance": formatAccountBalance,
			"FormatTransactionID":  formatTransactionID,
			"ShouldDisplayTime":    shouldDisplayTime,
			"IncludesID":           includesID,
			"TimeNow":              timeNow,
			"DaysBefore":           daysBefore,
			"SortCategories":       sortCategories,
			"ContainPrefix":        containPrefix,
		}).
		ParseFiles(templates...)
	if err != nil {
		log.Fatal("parse template file error:", err.Error())
	}

	return &View{
		Template: t,
		Layout:   "base",
	}
}

func NewEmailView(templateName string) (*template.Template, error) {
	templates := "web/template/email/" + templateName + ".html"

	t, err := template.New("").
		Funcs(template.FuncMap{
			"ArrToSting":         arrToSting,
			"TagsToString":       tagsToString,
			"TagsToSearchString": tagsToSearchString,
			"Add":                add,
			"Minus":              minus,
			"N":                  n,
			"IDToString":         idToString,
			"FormatTime":         formatTime,
			"ShouldDisplayTime":  shouldDisplayTime,
			"IncludesID":         includesID,
		}).
		ParseFiles(templates)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// Render is used to render the view with the predefined layout.
func (v *View) Render(w http.ResponseWriter, r *http.Request, yield interface{}, ErrorMessages []string) {
	w.Header().Set("Content-Type", "text/html")

	var vd Data
	vd.User.ID = r.Header.Get("userID")
	admin, err := strconv.ParseBool(r.Header.Get("admin"))
	if err != nil {
		vd.User.Admin = false
	} else {
		vd.User.Admin = admin
	}
	vd.ErrorMessages = ErrorMessages
	vd.Yield = yield
	vd.Messages.Success = flash.GetFlash(w, r, constant.Flash.Success)
	vd.Messages.Info = flash.GetFlash(w, r, constant.Flash.Info)

	v.Template.ExecuteTemplate(w, v.Layout, vd)
}

// Success renders the self defined success message.
func (v *View) Success(w http.ResponseWriter, r *http.Request, yield interface{}, message string) {
	w.Header().Set("Content-Type", "text/html")

	var vd Data
	vd.User.ID = r.Header.Get("userID")
	admin, err := strconv.ParseBool(r.Header.Get("admin"))
	if err != nil {
		vd.User.Admin = false
	} else {
		vd.User.Admin = admin
	}
	vd.Yield = yield
	vd.Messages.Success = message

	v.Template.ExecuteTemplate(w, v.Layout, vd)
}

// Error renders the self defined error message.
func (v *View) Error(w http.ResponseWriter, r *http.Request, yield interface{}, err error) {
	w.Header().Set("Content-Type", "text/html")

	var vd Data
	error, ok := err.(e.Error)
	if ok {
		vd.ErrorMessages = []string{error.Message()}
	} else {
		vd.ErrorMessages = []string{"Sorry, something went wrong. Please try again later."}
	}
	vd.User.ID = r.Header.Get("userID")
	admin, err := strconv.ParseBool(r.Header.Get("admin"))
	if err != nil {
		vd.User.Admin = false
	} else {
		vd.User.Admin = admin
	}
	vd.Yield = yield

	v.Template.ExecuteTemplate(w, v.Layout, vd)
}

// layoutFiles returns a slice of strings representing
// the layout files used in our application.
func layoutFiles() []string {
	files, err := filepath.Glob(layoutDir + "*" + templateExt)
	if err != nil {
		panic(err)
	}
	return files
}
