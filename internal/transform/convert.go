package transform

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"
)

// ConvertSwagger converts a Swagger 2.0 spec to OpenAPI 3.0 format.
func ConvertSwagger(data []byte) ([]byte, error) {
	var spec map[string]interface{}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing spec: %w", err)
	}

	ver, _ := spec["swagger"].(string)
	if !strings.HasPrefix(ver, "2.") {
		return nil, fmt.Errorf("not a Swagger 2.0 spec (found %q)", ver)
	}

	out := make(map[string]interface{})
	out["openapi"] = "3.0.3"

	if info, ok := spec["info"]; ok {
		out["info"] = info
	}

	convertServers(spec, out)
	convertPaths(spec, out)
	convertDefinitions(spec, out)
	convertSecurityDefinitions(spec, out)

	if security, ok := spec["security"]; ok {
		out["security"] = security
	}
	if tags, ok := spec["tags"]; ok {
		out["tags"] = tags
	}
	if extDoc, ok := spec["externalDocs"]; ok {
		out["externalDocs"] = extDoc
	}

	return yaml.Marshal(out)
}

func convertServers(spec, out map[string]interface{}) {
	host, _ := spec["host"].(string)
	basePath, _ := spec["basePath"].(string)
	schemes, _ := spec["schemes"].([]interface{})

	if host == "" {
		return
	}

	if basePath == "" {
		basePath = "/"
	}

	var servers []interface{}
	if len(schemes) == 0 {
		schemes = []interface{}{"https"}
	}
	for _, s := range schemes {
		scheme, _ := s.(string)
		if scheme == "" {
			continue
		}
		servers = append(servers, map[string]interface{}{
			"url": fmt.Sprintf("%s://%s%s", scheme, host, basePath),
		})
	}
	if len(servers) > 0 {
		out["servers"] = servers
	}
}

func convertPaths(spec, out map[string]interface{}) {
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		return
	}

	consumes, _ := spec["consumes"].([]interface{})
	produces, _ := spec["produces"].([]interface{})

	converted := make(map[string]interface{})
	for path, pathItem := range paths {
		pi, ok := pathItem.(map[string]interface{})
		if !ok {
			converted[path] = pathItem
			continue
		}
		newPI := make(map[string]interface{})
		for method, op := range pi {
			opMap, ok := op.(map[string]interface{})
			if !ok {
				newPI[method] = op
				continue
			}
			newPI[method] = convertOperation(opMap, consumes, produces)
		}
		converted[path] = newPI
	}
	out["paths"] = converted
}

func convertOperation(op map[string]interface{}, globalConsumes, globalProduces []interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range op {
		switch k {
		case "parameters", "responses", "consumes", "produces":
			// handled separately
		default:
			result[k] = v
		}
	}

	opConsumes, ok := op["consumes"].([]interface{})
	if !ok {
		opConsumes = globalConsumes
	}
	opProduces, ok := op["produces"].([]interface{})
	if !ok {
		opProduces = globalProduces
	}

	convertParams(op, result, opConsumes)
	convertResponses(op, result, opProduces)

	return result
}

func convertParams(op, result map[string]interface{}, consumes []interface{}) {
	params, ok := op["parameters"].([]interface{})
	if !ok {
		return
	}

	var newParams []interface{}
	var bodySchema map[string]interface{}

	for _, p := range params {
		param, ok := p.(map[string]interface{})
		if !ok {
			continue
		}

		in, _ := param["in"].(string)
		if in == "body" {
			if s, ok := param["schema"]; ok {
				if converted, ok := convertRef(s).(map[string]interface{}); ok {
					bodySchema = converted
				}
			}
			continue
		}

		if in == "formData" {
			// form data params become part of requestBody
			continue
		}

		newParam := make(map[string]interface{})
		newParam["name"] = param["name"]
		newParam["in"] = in
		if desc, ok := param["description"]; ok {
			newParam["description"] = desc
		}
		if req, ok := param["required"]; ok {
			newParam["required"] = req
		}

		schema := make(map[string]interface{})
		if t, ok := param["type"]; ok {
			schema["type"] = t
		}
		if f, ok := param["format"]; ok {
			schema["format"] = f
		}
		if e, ok := param["enum"]; ok {
			schema["enum"] = e
		}
		if def, ok := param["default"]; ok {
			schema["default"] = def
		}
		if items, ok := param["items"]; ok {
			schema["items"] = convertRef(items)
		}
		newParam["schema"] = schema

		newParams = append(newParams, newParam)
	}

	if len(newParams) > 0 {
		result["parameters"] = newParams
	}

	if bodySchema != nil {
		mediaType := "application/json"
		if len(consumes) > 0 {
			if ct, ok := consumes[0].(string); ok {
				mediaType = ct
			}
		}
		result["requestBody"] = map[string]interface{}{
			"content": map[string]interface{}{
				mediaType: map[string]interface{}{
					"schema": bodySchema,
				},
			},
		}
	}
}

func convertResponses(op, result map[string]interface{}, produces []interface{}) {
	responses, ok := op["responses"].(map[string]interface{})
	if !ok {
		return
	}

	mediaType := "application/json"
	if len(produces) > 0 {
		if ct, ok := produces[0].(string); ok {
			mediaType = ct
		}
	}

	newResponses := make(map[string]interface{})
	for code, resp := range responses {
		r, ok := resp.(map[string]interface{})
		if !ok {
			newResponses[code] = resp
			continue
		}

		newResp := make(map[string]interface{})
		desc, _ := r["description"].(string)
		if desc == "" {
			desc = "Response"
		}
		newResp["description"] = desc

		if schema, ok := r["schema"]; ok {
			newResp["content"] = map[string]interface{}{
				mediaType: map[string]interface{}{
					"schema": convertRef(schema),
				},
			}
		}

		if headers, ok := r["headers"].(map[string]interface{}); ok {
			newHeaders := make(map[string]interface{})
			for name, h := range headers {
				hMap, ok := h.(map[string]interface{})
				if !ok {
					continue
				}
				newHeader := make(map[string]interface{})
				if d, ok := hMap["description"]; ok {
					newHeader["description"] = d
				}
				schema := make(map[string]interface{})
				if t, ok := hMap["type"]; ok {
					schema["type"] = t
				}
				if f, ok := hMap["format"]; ok {
					schema["format"] = f
				}
				newHeader["schema"] = schema
				newHeaders[name] = newHeader
			}
			if len(newHeaders) > 0 {
				newResp["headers"] = newHeaders
			}
		}

		newResponses[code] = newResp
	}
	result["responses"] = newResponses
}

func convertDefinitions(spec, out map[string]interface{}) {
	defs, ok := spec["definitions"].(map[string]interface{})
	if !ok {
		return
	}

	components, _ := out["components"].(map[string]interface{})
	if components == nil {
		components = make(map[string]interface{})
	}

	schemas := make(map[string]interface{})
	for name, def := range defs {
		schemas[name] = convertRef(def)
	}
	components["schemas"] = schemas
	out["components"] = components
}

func convertSecurityDefinitions(spec, out map[string]interface{}) {
	secDefs, ok := spec["securityDefinitions"].(map[string]interface{})
	if !ok {
		return
	}

	components, _ := out["components"].(map[string]interface{})
	if components == nil {
		components = make(map[string]interface{})
	}

	secSchemes := make(map[string]interface{})
	for name, def := range secDefs {
		d, ok := def.(map[string]interface{})
		if !ok {
			continue
		}

		scheme := make(map[string]interface{})
		typ, _ := d["type"].(string)

		switch typ {
		case "basic":
			scheme["type"] = "http"
			scheme["scheme"] = "basic"
		case "apiKey":
			scheme["type"] = "apiKey"
			if n, ok := d["name"]; ok {
				scheme["name"] = n
			}
			if in, ok := d["in"]; ok {
				scheme["in"] = in
			}
		case "oauth2":
			scheme["type"] = "oauth2"
			flow, _ := d["flow"].(string)
			flows := make(map[string]interface{})
			flowDef := make(map[string]interface{})
			if authURL, ok := d["authorizationUrl"]; ok {
				flowDef["authorizationUrl"] = authURL
			}
			if tokenURL, ok := d["tokenUrl"]; ok {
				flowDef["tokenUrl"] = tokenURL
			}
			if scopes, ok := d["scopes"]; ok {
				flowDef["scopes"] = scopes
			} else {
				flowDef["scopes"] = map[string]interface{}{}
			}

			switch flow {
			case "implicit":
				flows["implicit"] = flowDef
			case "password":
				flows["password"] = flowDef
			case "application":
				flows["clientCredentials"] = flowDef
			case "accessCode":
				flows["authorizationCode"] = flowDef
			}
			scheme["flows"] = flows
		default:
			continue
		}

		if desc, ok := d["description"]; ok {
			scheme["description"] = desc
		}
		secSchemes[name] = scheme
	}

	if len(secSchemes) > 0 {
		components["securitySchemes"] = secSchemes
		out["components"] = components
	}
}

// convertRef converts $ref paths from Swagger 2.0 format to OpenAPI 3.0 format.
func convertRef(v interface{}) interface{} {
	m, ok := v.(map[string]interface{})
	if !ok {
		return v
	}

	result := make(map[string]interface{})
	for k, val := range m {
		switch k {
		case "$ref":
			if s, ok := val.(string); ok {
				result["$ref"] = strings.Replace(s, "#/definitions/", "#/components/schemas/", 1)
			} else {
				result[k] = val
			}
		case "properties":
			if props, ok := val.(map[string]interface{}); ok {
				newProps := make(map[string]interface{})
				for pName, pVal := range props {
					newProps[pName] = convertRef(pVal)
				}
				result[k] = newProps
			} else {
				result[k] = val
			}
		case "items":
			result[k] = convertRef(val)
		case "additionalProperties":
			result[k] = convertRef(val)
		default:
			result[k] = val
		}
	}
	return result
}
