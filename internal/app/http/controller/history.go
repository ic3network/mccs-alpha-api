package controller

import (
	"sync"

	"github.com/gorilla/mux"
)

var HistoryHandler = newHistoryHandler()

type historyHandler struct {
	once *sync.Once
}

func newHistoryHandler() *historyHandler {
	return &historyHandler{
		once: new(sync.Once),
	}
}

func (h *historyHandler) RegisterRoutes(
	public *mux.Router,
	private *mux.Router,
	adminPublic *mux.Router,
	adminPrivate *mux.Router,
) {
	h.once.Do(func() {
		// adminPrivate.Path("/history/{id}").HandlerFunc(h.historyPage()).Methods("GET")
	})
}

// func (h *adminHistoryHandler) historyPage() func(http.ResponseWriter, *http.Request) {
// 	t := template.NewView("admin/history")
// 	type formData struct {
// 		DateFrom string
// 		DateTo   string
// 		Page     int
// 	}
// 	type response struct {
// 		FormData     formData
// 		TotalPages   int
// 		Balance      float64
// 		Transactions []*types.Transfer
// 		Email        string
// 		EntityID     string
// 	}
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		vars := mux.Vars(r)
// 		bID := vars["id"]
// 		q := r.URL.Query()

// 		page, err := strconv.Atoi(q.Get("page"))
// 		if err != nil {
// 			l.Logger.Error("controller.AdminHistory.HistoryPage failed", zap.Error(err))
// 			t.Error(w, r, nil, err)
// 			return
// 		}

// 		f := formData{
// 			DateFrom: q.Get("date-from"),
// 			DateTo:   q.Get("date-to"),
// 			Page:     page,
// 		}
// 		user, err := UserHandler.FindByEntityID(bID)
// 		if err != nil {
// 			l.Logger.Error("controller.AdminHistory.HistoryPage failed", zap.Error(err))
// 			t.Error(w, r, nil, err)
// 			return
// 		}
// 		res := response{FormData: f, EntityID: bID, Email: user.Email}

// 		// Get the account balance.
// 		account, err := logic.Account.FindByEntityID(bID)
// 		if err != nil {
// 			l.Logger.Error("controller.AdminHistory.HistoryPage failed", zap.Error(err))
// 			t.Error(w, r, nil, err)
// 			return
// 		}
// 		res.Balance = account.Balance

// 		// Get the recent transactions.
// 		transactions, totalPages, err := logic.Transfer.FindInRange(
// 			account.ID,
// 			util.ParseTime(f.DateFrom),
// 			util.ParseTime(f.DateTo),
// 			page,
// 		)
// 		if err != nil {
// 			l.Logger.Error("controller.AdminHistory.HistoryPage failed", zap.Error(err))
// 			t.Error(w, r, nil, err)
// 			return
// 		}
// 		res.Transactions = transactions
// 		res.TotalPages = totalPages

// 		t.Render(w, r, res, nil)
// 	}
// }