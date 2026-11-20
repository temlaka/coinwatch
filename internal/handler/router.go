package handler

import (
	"github.com/temlaka/coinwatch/docs"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func Routes(authH *AuthHandler, cryptoH *CryptoHandler, scheduleH *ScheduleHandler, secret string) http.Handler {

	muxAuth := http.NewServeMux()

	muxAuth.HandleFunc("GET /crypto", cryptoH.GetCryptos)
	muxAuth.HandleFunc("POST /crypto", cryptoH.AddCrypto)
	muxAuth.HandleFunc("GET /crypto/{symbol}", cryptoH.GetCryptoInfo)
	muxAuth.HandleFunc("PUT /crypto/{symbol}/refresh", cryptoH.UpdateCryptoInfo)
	muxAuth.HandleFunc("GET /crypto/{symbol}/history", cryptoH.GetCryptoHistory)
	muxAuth.HandleFunc("GET /crypto/{symbol}/stats", cryptoH.GetStatsCrypto)
	muxAuth.HandleFunc("DELETE /crypto/{symbol}", cryptoH.DeleteCrypto)

	muxAuth.HandleFunc("GET /schedule", scheduleH.GetSettings)
	muxAuth.HandleFunc("PUT /schedule", scheduleH.SetSettings)
	muxAuth.HandleFunc("POST /schedule/trigger", scheduleH.Update)

	protected := AuthCheck(secret, muxAuth)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /auth/register", authH.Register)
	mux.HandleFunc("POST /auth/login", authH.Login)

	mux.Handle("/schedule", protected)
	mux.Handle("/schedule/", protected)
	mux.Handle("/crypto", protected)
	mux.Handle("/crypto/", protected)

	mux.Handle("GET /docs", http.RedirectHandler("/docs/index.html", http.StatusMovedPermanently))
	mux.Handle("GET /docs/openapi.yaml", http.StripPrefix("/docs/", docs.SpecHandler()))
	mux.Handle("GET /docs/", httpSwagger.Handler(httpSwagger.URL("/docs/openapi.yaml")))

	return mux

}
