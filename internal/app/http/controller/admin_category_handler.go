package controller

import (
	"errors"
	"net/http"
	"strconv"

	"sync"

	"github.com/gorilla/mux"
	"github.com/ic3network/mccs-alpha-api/internal/app/api"
	"github.com/ic3network/mccs-alpha-api/internal/app/logic"
	"github.com/ic3network/mccs-alpha-api/internal/app/types"
	"github.com/ic3network/mccs-alpha-api/internal/pkg/l"
	"github.com/ic3network/mccs-alpha-api/internal/pkg/template"
	"github.com/ic3network/mccs-alpha-api/internal/pkg/utils"
	"go.uber.org/zap"
)

type categoryHandler struct {
	once *sync.Once
}

var CategoryHandler = newCategoryHandler()

func newCategoryHandler() *categoryHandler {
	return &categoryHandler{
		once: new(sync.Once),
	}
}

func (handler *categoryHandler) RegisterRoutes(
	public *mux.Router,
	private *mux.Router,
	adminPublic *mux.Router,
	adminPrivate *mux.Router,
) {
	handler.once.Do(func() {
		adminPrivate.Path("/categories").HandlerFunc(handler.createCategory()).Methods("POST")
		public.Path("/categories").HandlerFunc(handler.searchCategory()).Methods("GET")
		adminPrivate.Path("/categories/{id}").HandlerFunc(handler.updateCategory()).Methods("PATCH")
		adminPrivate.Path("/categories/{id}").HandlerFunc(handler.deleteCategory()).Methods("DELETE")

		adminPrivate.Path("/categories").HandlerFunc(handler.SearchCategories()).Methods("GET")
	})
}

func (handler *categoryHandler) Update(categories []string) {
	err := logic.Category.Create(categories...)
	if err != nil {
		l.Logger.Error("[Error] CategoryHandler.Update failed:", zap.Error(err))
	}
}

func (handler *categoryHandler) createCategory() func(http.ResponseWriter, *http.Request) {
	type respond struct {
		Data *types.CategoryRespond `json:"data"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		req, errs := types.NewAdminCreateCategoryReqBody(r)
		if len(errs) > 0 {
			api.Respond(w, r, http.StatusBadRequest, errs)
			return
		}

		_, err := logic.Category.FindByName(req.Name)
		if err == nil {
			api.Respond(w, r, http.StatusBadRequest, errors.New("Category already exists."))
			return
		}

		created, err := logic.Category.CreateOne(req.Name)
		if err != nil {
			l.Logger.Error("[Error] CategoryHandler.createCategory failed:", zap.Error(err))
			api.Respond(w, r, http.StatusInternalServerError, err)
			return
		}

		api.Respond(w, r, http.StatusOK, respond{Data: &types.CategoryRespond{
			ID:   created.ID.Hex(),
			Name: created.Name,
		}})
	}
}

func (handler *categoryHandler) searchCategory() func(http.ResponseWriter, *http.Request) {
	type meta struct {
		NumberOfResults int `json:"numberOfResults"`
		TotalPages      int `json:"totalPages"`
	}
	type respond struct {
		Data []string `json:"data"`
		Meta meta     `json:"meta"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := api.NewSearchCategoryQuery(r.URL.Query())
		if err != nil {
			l.Logger.Info("[Info] CategoryHandler.searchCategory failed:", zap.Error(err))
			api.Respond(w, r, http.StatusBadRequest, err)
			return
		}

		errs := query.Validate()
		if len(errs) > 0 {
			api.Respond(w, r, http.StatusBadRequest, errs)
			return
		}

		found, err := logic.Category.Search(query)
		if err != nil {
			l.Logger.Error("[Error] CategoryHandler.searchCategory failed:", zap.Error(err))
			api.Respond(w, r, http.StatusInternalServerError, err)
			return
		}

		api.Respond(w, r, http.StatusOK, respond{
			Data: utils.CategoryToNames(found.Categories),
			Meta: meta{
				TotalPages:      found.TotalPages,
				NumberOfResults: found.NumberOfResults,
			},
		})
	}
}

func (handler *categoryHandler) updateCategory() func(http.ResponseWriter, *http.Request) {
	type respond struct {
		Data *types.CategoryRespond `json:"data"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		req, errs := types.NewAdminUpdateCategoryReqBody(r)
		if len(errs) > 0 {
			api.Respond(w, r, http.StatusBadRequest, errs)
			return
		}

		_, err := logic.Category.FindByName(req.Name)
		if err == nil {
			api.Respond(w, r, http.StatusBadRequest, errors.New("Category already exists."))
			return
		}

		old, err := logic.Category.FindByIDString(req.ID)
		if err != nil {
			l.Logger.Error("[Error] CategoryHandler.updateCategory failed:", zap.Error(err))
			api.Respond(w, r, http.StatusInternalServerError, err)
			return
		}

		updated, err := logic.Category.FindOneAndUpdate(old.ID, &types.Category{Name: req.Name})
		if err != nil {
			l.Logger.Error("[Error] CategoryHandler.updateCategory failed:", zap.Error(err))
			api.Respond(w, r, http.StatusInternalServerError, err)
			return
		}

		go func() {
			err := logic.Entity.RenameCategory(old.Name, updated.Name)
			if err != nil {
				l.Logger.Error("[Error] CategoryHandler.updateCategory failed:", zap.Error(err))
				return
			}
		}()

		api.Respond(w, r, http.StatusOK, respond{Data: &types.CategoryRespond{
			ID:   updated.ID.Hex(),
			Name: updated.Name,
		}})
	}
}

func (handler *categoryHandler) deleteCategory() func(http.ResponseWriter, *http.Request) {
	type respond struct {
		Data *types.CategoryRespond `json:"data"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		req, errs := types.NewAdminDeleteCategoryReqBody(r)
		if len(errs) > 0 {
			api.Respond(w, r, http.StatusBadRequest, errs)
			return
		}

		deleted, err := logic.Category.FindOneAndDelete(req.ID)
		if err != nil {
			l.Logger.Error("[Error] CategoryHandler.deleteCategory failed:", zap.Error(err))
			api.Respond(w, r, http.StatusBadRequest, err)
			return
		}

		go func() {
			err := logic.Entity.DeleteCategory(deleted.Name)
			if err != nil {
				l.Logger.Error("[Error] logic.Entity.DeleteCategory failed:", zap.Error(err))
			}
		}()

		api.Respond(w, r, http.StatusOK, respond{Data: &types.CategoryRespond{
			ID:   deleted.ID.Hex(),
			Name: deleted.Name,
		}})
	}
}

// TO BE REMOVED

func (handler *categoryHandler) SaveCategories(categories []string) error {
	for _, category := range categories {
		err := logic.Category.Create(category)
		if err != nil {
			return err
		}
	}
	return nil
}

func (handler *categoryHandler) SearchCategories() func(http.ResponseWriter, *http.Request) {
	t := template.NewView("admin/admin-tags")
	type formData struct {
		Name string
		Page int
	}
	type response struct {
		FormData formData
		Result   *types.FindCategoryResult
	}
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil {
			l.Logger.Error("SearchCategories failed", zap.Error(err))
			t.Error(w, r, nil, err)
			return
		}

		f := formData{
			Name: r.URL.Query().Get("name"),
			Page: page,
		}
		res := response{FormData: f}

		if f.Name == "" {
			t.Render(w, r, res, []string{"Please enter the category name"})
			return
		}

		findResult, err := logic.Category.FindTags(f.Name, int64(f.Page))
		if err != nil {
			l.Logger.Error("SearchCategories failed", zap.Error(err))
			t.Error(w, r, res, err)
			return
		}
		res.Result = findResult

		t.Render(w, r, res, nil)
	}
}
