package handler

import (
	"context"
	"net/http"

	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/gorilla/mux"
	"go.uber.org/zap"

	cev2event "github.com/cloudevents/sdk-go/v2/event"
	cev2http "github.com/cloudevents/sdk-go/v2/protocol/http"
)

// Handler interface for the SinkHandler.
type Handler interface {
	Start() error
}

type SinkHandler struct {
	logger *zap.Logger
	events map[string]*cev2event.Event
	cache  map[string]*cev2event.Event
}

func NewSinkHandler(logger *zap.Logger) *SinkHandler {
	return &SinkHandler{
		logger: logger,
		events: make(map[string]*cev2event.Event),
	}
}

func (h *SinkHandler) Start() error {
	router := mux.NewRouter()
	router.HandleFunc("/events", h.StoreEvent).Methods(http.MethodPost)
	router.HandleFunc("/events/{eventID}", h.GetEvent).Methods(http.MethodGet)

	return http.ListenAndServe(":8080", router)
}

func (h *SinkHandler) StoreEvent(w http.ResponseWriter, r *http.Request) {
	event, err := extractCloudEventFromRequest(r)
	if err != nil {
		h.namedLogger().With().Error("failed to extract CloudEvent from request", zap.Error(err))
		e := writeResponse(w, http.StatusBadRequest, []byte(err.Error()))
		if e != nil {
			h.namedLogger().Error("failed to write response", zap.Error(e))
		}
		return
	}

	h.events[event.ID()] = event
	err = writeResponse(w, http.StatusNoContent, []byte(""))
	if err != nil {
		h.namedLogger().Error("failed to write response", zap.Error(err))
	}
}

func (h *SinkHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	eventID := mux.Vars(r)["eventID"]
	event, ok := h.events[eventID]
	if !ok {
		h.namedLogger().With().Error("event not found", zap.String("eventID", eventID))
		e := writeResponse(w, http.StatusNotFound, []byte("event not found"))
		if e != nil {
			h.namedLogger().Error("failed to write response", zap.Error(e))
		}
		return
	}

	respBody, err := event.MarshalJSON()
	if err != nil {
		h.namedLogger().With().Error("failed to marshal event", zap.Error(err))
		e := writeResponse(w, http.StatusInternalServerError, []byte(err.Error()))
		if e != nil {
			h.namedLogger().Error("failed to write response", zap.Error(e))
		}
		return
	}

	err = writeResponse(w, http.StatusOK, respBody)
	if err != nil {
		h.namedLogger().Error("failed to write response", zap.Error(err))
	}
}

func (h *SinkHandler) namedLogger() *zap.Logger {
	return h.logger.Named("sink-handler")
}

// extractCloudEventFromRequest converts an incoming CloudEvent request to an Event.
func extractCloudEventFromRequest(r *http.Request) (*cev2event.Event, error) {
	message := cev2http.NewMessageFromHttpRequest(r)
	defer func() { _ = message.Finish(nil) }()

	event, err := binding.ToEvent(context.Background(), message)
	if err != nil {
		return nil, err
	}

	return event, nil
}

// writeResponse writes the HTTP response given the status code and response body.
func writeResponse(writer http.ResponseWriter, statusCode int, respBody []byte) error {
	writer.WriteHeader(statusCode)

	if respBody == nil {
		return nil
	}
	_, err := writer.Write(respBody)
	return err
}
