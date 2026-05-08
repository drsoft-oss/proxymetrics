package api

import "net/http"

func configHandler(deploymentSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeProblem(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"deployment_secret": deploymentSecret,
		})
	}
}
