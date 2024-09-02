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
package types

import (
	"reflect"

	log "github.com/sirupsen/logrus"
)

type TemplateData struct {
	TemplateName string
	Code         string
	CodeText     string
	RequestID    string
	Format       string
	Time         string
	OriginalURI  string
	Namespace    string
	IngressName  string
	ServiceName  string
	ServicePort  string
}

func (data *TemplateData) Fields() log.Fields {
	v := reflect.ValueOf(*data)
	fields := make(log.Fields)

	for i := range v.NumField() {
		fields[v.Type().Field(i).Name] = v.Field(i).String()
	}

	return fields
}
