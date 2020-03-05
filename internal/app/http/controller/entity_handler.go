package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/mux"
	"github.com/ic3network/mccs-alpha-api/global/constant"
	"github.com/ic3network/mccs-alpha-api/internal/app/logic"
	"github.com/ic3network/mccs-alpha-api/internal/app/types"
	"github.com/ic3network/mccs-alpha-api/internal/pkg/api"
	"github.com/ic3network/mccs-alpha-api/internal/pkg/email"
	"github.com/ic3network/mccs-alpha-api/internal/pkg/l"
	"github.com/ic3network/mccs-alpha-api/util"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
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
		public.Path("/entities").HandlerFunc(b.searchEntity()).Methods("GET")
		public.Path("/entities/{searchEntityID}").HandlerFunc(b.getEntity()).Methods("GET")
		private.Path("/favorites").HandlerFunc(b.addToFavoriteEntities()).Methods("POST")
		private.Path("/send-email").HandlerFunc(b.sendEmailToEntity()).Methods("POST")

		private.Path("/entities/search/match-tags").HandlerFunc(b.searhMatchTags()).Methods("GET")
		private.Path("/api/entityStatus").HandlerFunc(b.entityStatus()).Methods("GET")
		private.Path("/api/getEntityName").HandlerFunc(b.getEntityName()).Methods("GET")
		private.Path("/api/tradingMemberStatus").HandlerFunc(b.tradingMemberStatus()).Methods("GET")
	})
}

func (handler *entityHandler) FindByID(entityID string) (*types.Entity, error) {
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

func (handler *entityHandler) FindByEmail(email string) (*types.Entity, error) {
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

func (handler *entityHandler) FindByUserID(uID string) (*types.Entity, error) {
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

func (handler *entityHandler) UpdateOffersAndWants(old *types.Entity, offers, wants []string) {
	if len(offers) == 0 && len(wants) == 0 {
		return
	}

	tagDifference := types.NewTagDifference(types.TagFieldToNames(old.Offers), offers, types.TagFieldToNames(old.Wants), wants)
	err := logic.Entity.UpdateTags(old.ID, tagDifference)
	if err != nil {
		l.Logger.Error("[Error] EntityHandler.UpdateOffersAndWants failed:", zap.Error(err))
		return
	}

	if util.IsAcceptedStatus(old.Status) {
		// User Update tags logic:
		// 	1. Update the tags collection only when the entity is in accepted status.
		err := TagHandler.UpdateOffers(tagDifference.NewAddedOffers)
		if err != nil {
			l.Logger.Error("[Error] EntityHandler.UpdateOffersAndWants failed:", zap.Error(err))
		}
		err = TagHandler.UpdateWants(tagDifference.NewAddedWants)
		if err != nil {
			l.Logger.Error("[Error] EntityHandler.UpdateOffersAndWants failed:", zap.Error(err))
		}
	}
}

func (handler *entityHandler) getSearchEntityQueryParams(q url.Values) (*types.SearchEntityQuery, error) {
	query, err := types.NewSearchEntityQuery(q)
	if err != nil {
		return nil, err
	}
	query.FavoriteEntities = handler.getFavoriteEntities(q.Get("querying_entity_id"))
	return query, nil
}

func (handler *entityHandler) getFavoriteEntities(entityID string) []primitive.ObjectID {
	entity, err := EntityHandler.FindByID(entityID)
	if err == nil {
		return entity.FavoriteEntities
	}
	return []primitive.ObjectID{}
}

func (handler *entityHandler) getQueryingEntityStatus(entityID string) string {
	entity, err := EntityHandler.FindByID(entityID)
	if err == nil {
		return entity.Status
	}
	return ""
}

func (handler *entityHandler) searchEntity() func(http.ResponseWriter, *http.Request) {
	type meta struct {
		NumberOfResults int `json:"numberOfResults"`
		TotalPages      int `json:"totalPages"`
	}
	type respond struct {
		Data []*types.SearchEntityRespond `json:"data"`
		Meta meta                         `json:"meta"`
	}
	toData := func(query *types.SearchEntityQuery, entities []*types.Entity) []*types.SearchEntityRespond {
		result := []*types.SearchEntityRespond{}
		queryingEntityStatus := handler.getQueryingEntityStatus(query.QueryingEntityID)
		for _, entity := range entities {
			result = append(result, types.NewSearchEntityRespond(entity, queryingEntityStatus, query.FavoriteEntities))
		}
		return result
	}
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := handler.getSearchEntityQueryParams(r.URL.Query())
		if err != nil {
			l.Logger.Info("[INFO] EntityHandler.searchEntity failed:", zap.Error(err))
			api.Respond(w, r, http.StatusBadRequest, err)
			return
		}

		errs := query.Validate()
		if len(errs) > 0 {
			api.Respond(w, r, http.StatusBadRequest, errs)
			return
		}

		found, err := logic.Entity.Find(query)
		if err != nil {
			l.Logger.Error("[Error] EntityHandler.searchEntity failed:", zap.Error(err))
			api.Respond(w, r, http.StatusBadRequest, err)
			return
		}

		api.Respond(w, r, http.StatusOK, respond{
			Data: toData(query, found.Entities),
			Meta: meta{
				TotalPages:      found.TotalPages,
				NumberOfResults: found.NumberOfResults,
			},
		})
	}
}

func (handler *entityHandler) getEntity() func(http.ResponseWriter, *http.Request) {
	type respond struct {
		Data *types.SearchEntityRespond `json:"data"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		searchEntity, err := logic.Entity.FindByStringID(vars["searchEntityID"])
		if err != nil {
			l.Logger.Info("[INFO] EntityHandler.getEntity failed:", zap.Error(err))
			api.Respond(w, r, http.StatusBadRequest, err)
			return
		}

		q := r.URL.Query()
		queryingEntityID := q.Get("querying_entity_id")
		queryingEntityStatus := handler.getQueryingEntityStatus(queryingEntityID)
		favoriteEntities := handler.getFavoriteEntities(queryingEntityID)

		api.Respond(w, r, http.StatusOK, respond{Data: types.NewSearchEntityRespond(searchEntity, queryingEntityStatus, favoriteEntities)})
	}
}

func (handler *entityHandler) addToFavoriteEntities() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AddToFavoriteReqBody
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&req)
		if err != nil {
			l.Logger.Info("[Info] EntityHandler.addToFavorite failed:", zap.Error(err))
			api.Respond(w, r, http.StatusBadRequest, err)
			return
		}

		errs := req.Validate()
		if len(errs) > 0 {
			api.Respond(w, r, http.StatusBadRequest, errs)
			return
		}

		err = logic.Entity.AddToFavoriteEntities(&req)
		if err != nil {
			l.Logger.Error("[Error] EntityHandler.addToFavorite failed:", zap.Error(err))
			api.Respond(w, r, http.StatusInternalServerError, err)
			return
		}

		api.Respond(w, r, http.StatusOK, nil)
	}
}

func (handler *entityHandler) checkEntityStatus(SenderEntity, ReceiverEntity *types.Entity) error {
	if !util.IsAcceptedStatus(SenderEntity.Status) {
		return errors.New("Sender does not have the correct status.")

	}
	if !util.IsAcceptedStatus(ReceiverEntity.Status) {
		return errors.New("Receiver does not have the correct status.")
	}
	return nil
}

func (handler *entityHandler) sendEmailToEntity() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := types.NewEmailReqBody(r)
		if err != nil {
			l.Logger.Info("[Info] EntityHandler.sendEmailToEntity failed:", zap.Error(err))
			api.Respond(w, r, http.StatusBadRequest, err)
			return
		}

		errs := req.Validate()
		if len(errs) > 0 {
			api.Respond(w, r, http.StatusBadRequest, errs)
			return
		}

		SenderEntity, err := handler.FindByID(req.SenderEntityID)
		if err != nil {
			l.Logger.Error("[Error] EntityHandler.sendEmailToEntity failed:", zap.Error(err))
			api.Respond(w, r, http.StatusInternalServerError, err)
			return
		}
		ReceiverEntity, err := handler.FindByID(req.ReceiverEntityID)
		if err != nil {
			l.Logger.Error("[Error] EntityHandler.sendEmailToEntity failed:", zap.Error(err))
			api.Respond(w, r, http.StatusInternalServerError, err)
			return
		}
		if !UserHandler.IsEntityBelongsToUser(req.SenderEntityID, r.Header.Get("userID")) {
			api.Respond(w, r, http.StatusForbidden, api.ErrPermissionDenied)
			return
		}
		err = handler.checkEntityStatus(SenderEntity, ReceiverEntity)
		if err != nil {
			api.Respond(w, r, http.StatusBadRequest, err)
			return
		}

		err = email.SendContactEntity(ReceiverEntity.EntityName, ReceiverEntity.Email, SenderEntity.EntityName, SenderEntity.Email, req.Body)
		if err != nil {
			l.Logger.Error("[Error] EntityHandler.sendEmailToEntity failed:", zap.Error(err))
			api.Respond(w, r, http.StatusInternalServerError, err)
			return
		}

		if viper.GetString("env") == "development" {
			type data struct {
				SenderEntityName   string `json:"sender_entity_name"`
				ReceiverEntityName string `json:"receiver_entity_name"`
				Body               string `json:"body"`
			}
			type respond struct {
				Data data `json:"data"`
			}
			api.Respond(w, r, http.StatusOK, respond{Data: data{
				SenderEntityName:   SenderEntity.EntityName,
				ReceiverEntityName: ReceiverEntity.EntityName,
				Body:               req.Body,
			}})
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// TO BE REMOVED

func (handler *entityHandler) searhMatchTags() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/entities/search?"+r.URL.Query().Encode(), http.StatusFound)
	}
}

func (handler *entityHandler) entityStatus() func(http.ResponseWriter, *http.Request) {
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

func (handler *entityHandler) getEntityName() func(http.ResponseWriter, *http.Request) {
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

func (handler *entityHandler) tradingMemberStatus() func(http.ResponseWriter, *http.Request) {
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
