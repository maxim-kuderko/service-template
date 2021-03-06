package main

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/julienschmidt/httprouter"
	gs "github.com/maxim-kuderko/graceful-shutdown"
	"github.com/maxim-kuderko/service-template/internal/initializers"
	"github.com/maxim-kuderko/service-template/internal/repositories/primary"
	"github.com/maxim-kuderko/service-template/internal/service"
	"github.com/maxim-kuderko/service-template/pkg/requests"
	"github.com/maxim-kuderko/service-template/pkg/responses"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"go.uber.org/fx"
	"net/http"
)

func main() {
	go fx.New(
		fx.NopLogger,
		fx.Provide(
			initializers.NewConfig,
			primary.NewCachedDB,
			service.NewService,
			newHandler,
			router,
		),
		fx.Invoke(webserver),
	)
	gs.WaitForGrace()
}

func router(h *handler) *httprouter.Router {
	router := httprouter.New()
	router.HandlerFunc(http.MethodPost, `/get`, h.Get)
	return router
}

func webserver(r *httprouter.Router, v *viper.Viper) {
	http.ListenAndServe(fmt.Sprintf(`:%s`, v.GetString(`HTTP_SERVER_PORT`)), r)
}

type handler struct {
	s *service.Service
}

func newHandler(s *service.Service) *handler {
	return &handler{
		s: s,
	}
}

func (h *handler) Get(w http.ResponseWriter, r *http.Request) {
	var req requests.Get
	if parser(w, r, &req) != nil {
		return
	}
	resp, err := h.s.Get(req)
	response(w, resp, err)
}

func parser(w http.ResponseWriter, r *http.Request, req requests.BaseRequester) error {
	req.WithContext(r.Context())
	err := jsoniter.ConfigFastest.NewDecoder(r.Body).Decode(&req)
	defer r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		jsoniter.ConfigFastest.NewEncoder(w).Encode(err)
	}
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		jsoniter.ConfigFastest.NewEncoder(w).Encode(err)
	}
	return err
}

func response(w http.ResponseWriter, resp responses.BaseResponser, err error) {
	w.Header().Set(`Content-Type`, `application/json`)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsoniter.ConfigFastest.NewEncoder(w).Encode(err)
		return
	}
	w.WriteHeader(resp.ResponseStatusCode())
	if err := jsoniter.ConfigFastest.NewEncoder(w).Encode(resp); err != nil {
		w.WriteHeader(fasthttp.StatusInternalServerError)
		jsoniter.ConfigFastest.NewEncoder(w).Encode(err)
		return
	}
	return
}
