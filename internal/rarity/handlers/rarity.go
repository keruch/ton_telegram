package handlers

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/keruch/ton_telegram/internal/rarity/application"
	"net/http"
	"strconv"
	"time"
)

type RarityHandler struct {
	service *application.RarityService
}

func NewRarityHandler(service *application.RarityService) *RarityHandler {
	return &RarityHandler{service: service}
}

func (h *RarityHandler) Attach(r *mux.Router) {
	r.Methods(http.MethodGet).PathPrefix("/health").HandlerFunc(getHealth)
	r.Methods(http.MethodGet).PathPrefix("/rarity/{id}").HandlerFunc(h.getRarity)
}

func getHealth(writer http.ResponseWriter, req *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func (h *RarityHandler) getRarity(writer http.ResponseWriter, req *http.Request) {
	rawId, ok := mux.Vars(req)["id"]
	if !ok {
		WriteResponse(writer, NewResponse("InternalServerError", 500))
		return
	}
	id, err := strconv.Atoi(rawId)
	if err != nil {
		WriteResponse(writer, NewResponse(err.Error(), 400))
		return
	}
	rarity, err := h.service.GetRarity(id)
	if errors.Is(err, application.ErrIdNotFound) {
		WriteResponse(writer, NewResponse(err.Error(), 404))
		return
	}

	WriteResponse(writer, NewResponse(rarity, 200))
}

func WriteResponse(writer http.ResponseWriter, resp Response) {
	rawResp, err := json.Marshal(resp)
	if err != nil {
		writer.WriteHeader(500)
		return
	}
	writer.WriteHeader(resp.Status)
	_, err = writer.Write(rawResp)
	if err != nil {
		writer.WriteHeader(500)
		return
	}
}

func NewResponse(payload interface{}, status int) Response {
	return Response{
		Status:  status,
		Time:    time.Now(),
		Payload: payload,
	}
}

type Response struct {
	Status  int         `json:"status"`
	Time    time.Time   `json:"time"`
	Payload interface{} `json:"payload"`
}
