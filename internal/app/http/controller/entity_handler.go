package controller

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sync"

	"strconv"

	"github.com/gorilla/mux"
	"github.com/ic3network/mccs-alpha-api/global/constant"
	"github.com/ic3network/mccs-alpha-api/internal/app/logic"
	"github.com/ic3network/mccs-alpha-api/internal/app/types"
	"github.com/ic3network/mccs-alpha-api/internal/pkg/api"
	"github.com/ic3network/mccs-alpha-api/internal/pkg/email"
	"github.com/ic3network/mccs-alpha-api/internal/pkg/l"
	"github.com/ic3network/mccs-alpha-api/internal/pkg/validate"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"github.com/ic3network/mccs-alpha-api/internal/pkg/util"
)

type entityHandler struct {
	once *sync.Once
}

var EntityHandler = newEntityHandler()

func newEntityHandler() *entityHandler {
	return &entityHandler{
		once: new(sync.Once),
	}
}

func (b *entityHandler) RegisterRoutes(
	public *mux.Router,
	private *mux.Router,
	adminPublic *mux.Router,
	adminPrivate *mux.Router,
) {
	b.once.Do(func() {
		public.Path("/api/v1/entities").HandlerFunc(b.searchEntity()).Methods("GET")

		private.Path("/entities/search/match-tags").HandlerFunc(b.searhMatchTags()).Methods("GET")
		private.Path("/api/entityStatus").HandlerFunc(b.entityStatus()).Methods("GET")
		private.Path("/api/getEntityName").HandlerFunc(b.getEntityName()).Methods("GET")
		private.Path("/api/tradingMemberStatus").HandlerFunc(b.tradingMemberStatus()).Methods("GET")
		private.Path("/api/contactEntity").HandlerFunc(b.contactEntity()).Methods("POST")
	})
}

func (b *entityHandler) FindByID(entityID string) (*types.Entity, error) {
	objID, err := primitive.ObjectIDFromHex(entityID)
	if err != nil {
		return nil, err
	}
	entity, err := logic.Entity.FindByID(objID)
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func (b *entityHandler) FindByEmail(email string) (*types.Entity, error) {
	user, err := logic.User.FindByEmail(email)
	if err != nil {
		return nil, err
	}
	bs, err := logic.Entity.FindByID(user.Entities[0])
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func (b *entityHandler) FindByUserID(uID string) (*types.Entity, error) {
	user, err := UserHandler.FindByID(uID)
	if err != nil {
		return nil, err
	}
	bs, err := logic.Entity.FindByID(user.Entities[0])
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func getSearchEntityQuertParams(q url.Values) (*types.SearchEntityQuery, error) {
	page, err := strconv.Atoi(q.Get("page"))
	if err != nil {
		return nil, err
	}
	pageSize, err := strconv.Atoi(q.Get("page_size"))
	if err != nil {
		return nil, err
	}
	return &types.SearchEntityQuery{
		Page:          page,
		PageSize:      pageSize,
		Category:      q.Get("category"),
		Offers:        util.ToSearchTags(q.Get("offers")),
		Wants:         util.ToSearchTags(q.Get("wants")),
		TaggedSince:   util.ParseTime(q.Get("tagged_since")),
		FavoritesOnly: q.Get("favorites_only") == "true",
	}, nil
}

func getSearchCriteria(query *types.SearchEntityQuery, favoriteEntities []primitive.ObjectID) *types.SearchCriteria {
	return &types.SearchCriteria{
		Page:             query.Page,
		PageSize:         query.PageSize,
		Category:         query.Category,
		Offers:           query.Offers,
		Wants:            query.Wants,
		TaggedSince:      query.TaggedSince,
		FavoritesOnly:    query.FavoritesOnly,
		FavoriteEntities: favoriteEntities,
		Statuses: []string{
			constant.Entity.Accepted,
			constant.Trading.Pending,
			constant.Trading.Accepted,
			constant.Trading.Rejected,
		},
	}
}

func (b *entityHandler) searchEntity() func(http.ResponseWriter, *http.Request) {
	type data struct {
		ID                 string   `json:"id"`
		EntityName         string   `json:"entityName"`
		EntityPhone        string   `json:"entityPhone"`
		IncType            string   `json:"incType"`
		CompanyNumber      string   `json:"companyNumber"`
		Website            string   `json:"website"`
		Turnover           int      `json:"turnover"`
		Description        string   `json:"description"`
		LocationAddress    string   `json:"locationAddress"`
		LocationCity       string   `json:"locationCity"`
		LocationRegion     string   `json:"locationRegion"`
		LocationPostalCode string   `json:"locationPostalCode"`
		LocationCountry    string   `json:"locationCountry"`
		Status             string   `json:"status"`
		Offers             []string `json:"offers"`
		Wants              []string `json:"wants"`
		IsFavorite         bool     `json:"isFavorite"`
	}
	type meta struct {
		NumberOfResults int `json:"numberOfResults"`
		TotalPages      int `json:"totalPages"`
	}
	type respond struct {
		Data []*data `json:"data"`
		Meta meta    `json:"meta"`
	}
	toData := func(entities []*types.Entity, favorites []primitive.ObjectID) []*data {
		result := []*data{}
		for _, entity := range entities {
			isFavorite := util.ContainID(favorites, entity.ID)
			result = append(result, &data{
				ID:                 entity.ID.Hex(),
				EntityName:         entity.EntityName,
				EntityPhone:        entity.EntityPhone,
				IncType:            entity.IncType,
				CompanyNumber:      entity.CompanyNumber,
				Website:            entity.Website,
				Turnover:           entity.Turnover,
				Description:        entity.Description,
				LocationAddress:    entity.LocationAddress,
				LocationCity:       entity.LocationCity,
				LocationRegion:     entity.LocationRegion,
				LocationPostalCode: entity.LocationPostalCode,
				LocationCountry:    entity.LocationCountry,
				Status:             entity.Status,
				Offers:             util.GetTagNames(entity.Offers),
				Wants:              util.GetTagNames(entity.Wants),
				IsFavorite:         isFavorite,
			})
		}
		return result
	}
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := getSearchEntityQuertParams(r.URL.Query())
		if err != nil {
			l.Logger.Info("[INFO] EntityHandler.searchEntity failed:", zap.Error(err))
			api.Respond(w, r, http.StatusBadRequest, err)
			return
		}

		errs := validate.SearchBusiness(query)
		if len(errs) > 0 {
			api.Respond(w, r, http.StatusBadRequest, errs)
			return
		}

		var favoriteEntities []primitive.ObjectID
		user, err := UserHandler.FindByID(r.Header.Get("userID"))
		if err == nil {
			favoriteEntities = user.FavoriteEntities
		}
		criteria := getSearchCriteria(query, favoriteEntities)

		found, err := logic.Entity.Find(criteria)
		if err != nil {
			l.Logger.Error("[Error] EntityHandler.searchEntity failed:", zap.Error(err))
			api.Respond(w, r, http.StatusBadRequest, err)
			return
		}

		api.Respond(w, r, http.StatusOK, respond{
			Data: toData(found.Entities, favoriteEntities),
			Meta: meta{
				TotalPages:      found.TotalPages,
				NumberOfResults: found.NumberOfResults,
			},
		})
	}
}

func (b *entityHandler) contactEntity() func(http.ResponseWriter, *http.Request) {
	type request struct {
		EntityID string `json:"id"`
		Body     string `json:"body"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request

		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&req)
		if err != nil {
			l.Logger.Error("ContactEntity failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Something went wrong. Please try again later."))
			return
		}

		user, err := UserHandler.FindByID(r.Header.Get("userID"))
		if err != nil {
			l.Logger.Error("ContactEntity failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Something went wrong. Please try again later."))
			return
		}

		entityOwner, err := UserHandler.FindByEntityID(req.EntityID)
		if err != nil {
			l.Logger.Error("ContactEntity failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Something went wrong. Please try again later."))
			return
		}

		receiver := entityOwner.FirstName + " " + entityOwner.LastName
		replyToName := user.FirstName + " " + user.LastName
		err = email.SendContactEntity(receiver, entityOwner.Email, replyToName, user.Email, req.Body)
		if err != nil {
			l.Logger.Error("ContactEntity failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Something went wrong. Please try again later."))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (b *entityHandler) searhMatchTags() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/entities/search?"+r.URL.Query().Encode(), http.StatusFound)
	}
}

func (b *entityHandler) entityStatus() func(http.ResponseWriter, *http.Request) {
	type response struct {
		Status string `json:"status"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var entity *types.Entity
		var err error

		q := r.URL.Query()

		if q.Get("entity_id") != "" {
			objID, err := primitive.ObjectIDFromHex(q.Get("entity_id"))
			if err != nil {
				l.Logger.Error("EntityHandler.entityStatus failed", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			entity, err = logic.Entity.FindByID(objID)
			if err != nil {
				l.Logger.Error("EntityHandler.entityStatus failed", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else {
			entity, err = EntityHandler.FindByUserID(r.Header.Get("userID"))
			if err != nil {
				l.Logger.Error("EntityHandler.entityStatus failed", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		res := &response{Status: entity.Status}
		js, err := json.Marshal(res)
		if err != nil {
			l.Logger.Error("EntityHandler.entityStatus failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}
}

func (b *entityHandler) getEntityName() func(http.ResponseWriter, *http.Request) {
	type response struct {
		Name string
	}
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		user, err := logic.User.FindByEmail(q.Get("email"))
		if err != nil {
			l.Logger.Error("EntityHandler.getEntityName failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		entity, err := logic.Entity.FindByID(user.Entities[0])
		if err != nil {
			l.Logger.Error("EntityHandler.getEntityName failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res := response{Name: entity.EntityName}
		js, err := json.Marshal(res)
		if err != nil {
			l.Logger.Error("EntityHandler.getEntityName failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}
}

func (b *entityHandler) tradingMemberStatus() func(http.ResponseWriter, *http.Request) {
	type response struct {
		Self  bool `json:"self"`
		Other bool `json:"other"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		objID, err := primitive.ObjectIDFromHex(q.Get("entity_id"))
		if err != nil {
			l.Logger.Error("EntityHandler.tradingMemberStatus failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		other, err := logic.Entity.FindByID(objID)
		if err != nil {
			l.Logger.Error("EntityHandler.tradingMemberStatus failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		self, err := EntityHandler.FindByUserID(r.Header.Get("userID"))
		if err != nil {
			l.Logger.Error("EntityHandler.tradingMemberStatus failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res := &response{}
		if self.Status == constant.Trading.Accepted {
			res.Self = true
		}
		if other.Status == constant.Trading.Accepted {
			res.Other = true
		}
		js, err := json.Marshal(res)
		if err != nil {
			l.Logger.Error("EntityHandler.tradingMemberStatus failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}
}