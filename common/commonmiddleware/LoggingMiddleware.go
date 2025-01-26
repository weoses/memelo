package commonmiddleware

import (
	"log"

	echoServer "github.com/labstack/echo/v4"
	oapiEcho "github.com/oapi-codegen/runtime/strictmiddleware/echo"
)

func NewLoggingMiddleware() oapiEcho.StrictEchoMiddlewareFunc {
	return func(nextChain oapiEcho.StrictEchoHandlerFunc, operationID string) oapiEcho.StrictEchoHandlerFunc {
		return func(ctx echoServer.Context, request interface{}) (interface{}, error) {
			method := ctx.Request().Method
			url := ctx.Request().RequestURI

			log.Printf("Request  [%s] > %s %s", operationID, method, url)
			//log.Printf("RequestBody  [%s] > %s", operationID, render.Render(request))

			response, err := nextChain(ctx, request)
			if err != nil {
				log.Printf("Error    [%s] ! %v", operationID, err)
			} else {
				status := ctx.Response().Status

				log.Printf("Response [%s] < %d", operationID, status)
				//log.Printf("ResponseBody [%s] < %s", operationID, render.Render(response))
			}

			return response, err
		}
	}
}
