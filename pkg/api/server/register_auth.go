package server

import (
	"net/http"

	"github.com/containers/podman/v4/pkg/api/handlers/compat"
	"github.com/gorilla/mux"
)

func (s *APIServer) registerAuthHandlers(r *mux.Router) error {
	// swagger:operation POST /auth compat SystemAuth
	// ---
	//   summary: Check auth configuration
	//   tags:
	//    - system (compat)
	//   produces:
	//   - application/json
	//   parameters:
	//    - in: body
	//      name: authConfig
	//      description: Authentication to check
	//      schema:
	//        $ref: "#/definitions/AuthConfig"
	//   responses:
	//     200:
	//       $ref: "#/responses/SystemAuthResponse"
	//     500:
	//       $ref: "#/responses/InternalError"
	r.Handle(VersionedPath("/auth"), s.APIHandler(compat.Auth)).Methods(http.MethodPost)
	// Added non version path to URI to support docker non versioned paths
	r.Handle("/auth", s.APIHandler(compat.Auth)).Methods(http.MethodPost)
	return nil
}
