package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
)

// App *
type App struct {
	Router     *mux.Router
	Middleware *Middleware
	config     *Env
}

type shortenReq struct {
	URL                  string `json:"url" validate:"required"`
	ExpireationInMinutes int64  `json:"expiration_in_minutes" validate:"min=0"`
}

type shortlinkResp struct {
	Shortlink string `json:"shortlink"`
}

// Initialize *
func (a *App) Initialize(e *Env) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	a.Router = mux.NewRouter()
	a.Middleware = &Middleware{}
	a.initializeRouters()
	a.config = e
}

func (a *App) initializeRouters() {
	m := alice.New(a.Middleware.LoggingHandler, a.Middleware.RecoverHandler)
	a.Router.Handle("/api/shorten", m.ThenFunc(a.createShorten)).Methods("POST")
	a.Router.Handle("/api/info", m.ThenFunc(a.getInfo)).Methods("GET")
	a.Router.Handle("/api/{shortlink:[a-zA-Z0-9]{1,11}}", m.ThenFunc(a.getURLAndRedirect)).Methods("GET")
}

func (a *App) createShorten(w http.ResponseWriter, r *http.Request) {
	var req shortenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, StatusError{http.StatusBadRequest, fmt.Errorf("parse parameters failed %v", r.Body)})
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		respondWithError(w, StatusError{http.StatusBadRequest, fmt.Errorf("validate parameters failed %v", req)})
		return
	}

	defer r.Body.Close()

	s, err := a.config.S.Shorten(req.URL, req.ExpireationInMinutes)
	if err != nil {
		respondWithError(w, err)
	} else {
		respondWithJSON(w, http.StatusCreated, shortlinkResp{Shortlink: s})
	}
}

func (a *App) getInfo(w http.ResponseWriter, r *http.Request) {
	vals := r.URL.Query()
	s := vals.Get("shortlink")

	d, err := a.config.S.ShortlinkInfo(s)
	if err != nil {
		respondWithError(w, err)
	} else {
		respondWithJSON(w, http.StatusOK, d)
	}
}

func (a *App) getURLAndRedirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	u, err := a.config.S.Unshorten(vars["shortlink"])
	if err != nil {
		respondWithError(w, err)
	} else {
		http.Redirect(w, r, u, http.StatusTemporaryRedirect)
	}
}

// Run *
func (a *App) Run(addr string) {
	log.Fatal(http.ListenAndServe(addr, a.Router))
}

func respondWithError(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case Error:
		log.Printf("HTTP %d - %s", e.Status(), e)
		respondWithJSON(w, e.Status(), e.Error())
	default:
		respondWithJSON(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	resp, _ := json.Marshal(payload)

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(code)
	w.Write(resp)
}
