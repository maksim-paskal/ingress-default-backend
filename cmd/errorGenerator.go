/*
Copyright paskal.maksim@gmail.com
Licensed under the Apache License, Version 2.0 (the "License")
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

// function that will show template on code.
func errorGenerator(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	code := r.Form.Get("code")

	if len(code) == 0 {
		code = "200"
	}

	targetCodeInt, err := strconv.Atoi(code)
	if err != nil {
		targetCodeInt = 400

		log.Printf("%q looks like not a number.\n", code)
	}

	w.WriteHeader(targetCodeInt)
	_, err = w.Write([]byte(fmt.Sprintf("code=%d", targetCodeInt)))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
