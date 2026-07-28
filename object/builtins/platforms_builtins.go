package builtins

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"strconv"
	"strings"
	"zumbra/object"
)

var supabaseURL string
var supabaseKey string

func SupabaseConnectionBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseConnection(url, key). got=%d, want=2", len(args))
			}

			urlObj, ok1 := args[0].(*object.String)
			keyObj, ok2 := args[1].(*object.String)
			if !ok1 || !ok2 {
				return NewError("supabaseConnection(url, key) expects STRING, STRING")
			}

			supabaseURL = strings.TrimRight(urlObj.Value, "/")
			supabaseKey = keyObj.Value
			return &object.Null{}
		},
	}
}

func SupabaseSelectBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseSelect(table, selectQuery). got=%d, want=2", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			table, ok1 := args[0].(*object.String)
			selectQuery, ok2 := args[1].(*object.String)
			if !ok1 || !ok2 {
				return NewError("supabaseSelect(table, selectQuery) expects STRING, STRING")
			}

			query := "select=" + neturl.QueryEscape(selectQuery.Value)
			return supabaseRequest("GET", "/rest/v1/"+table.Value+"?"+query, nil, "")
		},
	}
}

func SupabaseQueryBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseQuery(table, queryString). got=%d, want=2", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			table, ok1 := args[0].(*object.String)
			queryString, ok2 := args[1].(*object.String)
			if !ok1 || !ok2 {
				return NewError("supabaseQuery(table, queryString) expects STRING, STRING")
			}

			qs := strings.TrimPrefix(queryString.Value, "?")
			return supabaseRequest("GET", "/rest/v1/"+table.Value+"?"+qs, nil, "")
		},
	}
}

func SupabaseInsertBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseInsert(table, payload). got=%d, want=2", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			table, ok := args[0].(*object.String)
			if !ok {
				return NewError("argument 1 to `supabaseInsert` must be STRING")
			}

			return supabaseRequest("POST", "/rest/v1/"+table.Value, args[1], "return=representation")
		},
	}
}

func SupabaseUpdateBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return NewError("wrong number of arguments, supabaseUpdate(table, filterQuery, payload). got=%d, want=3", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			table, ok1 := args[0].(*object.String)
			filterQuery, ok2 := args[1].(*object.String)
			if !ok1 || !ok2 {
				return NewError("supabaseUpdate(table, filterQuery, payload) expects STRING, STRING, ANY")
			}

			qs := strings.TrimPrefix(filterQuery.Value, "?")
			path := "/rest/v1/" + table.Value
			if qs != "" {
				path += "?" + qs
			}

			return supabaseRequest("PATCH", path, args[2], "return=representation")
		},
	}
}

func SupabaseDeleteBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseDelete(table, filterQuery). got=%d, want=2", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			table, ok1 := args[0].(*object.String)
			filterQuery, ok2 := args[1].(*object.String)
			if !ok1 || !ok2 {
				return NewError("supabaseDelete(table, filterQuery) expects STRING, STRING")
			}

			qs := strings.TrimPrefix(filterQuery.Value, "?")
			path := "/rest/v1/" + table.Value
			if qs != "" {
				path += "?" + qs
			}

			return supabaseRequest("DELETE", path, nil, "return=representation")
		},
	}
}

func SupabaseUpsertBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseUpsert(table, payload). got=%d, want=2", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			table, ok := args[0].(*object.String)
			if !ok {
				return NewError("argument 1 to `supabaseUpsert` must be STRING")
			}

			return supabaseRequest("POST", "/rest/v1/"+table.Value, args[1], "resolution=merge-duplicates,return=representation")
		},
	}
}

func SupabaseRpcBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseRpc(functionName, payload). got=%d, want=2", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			functionName, ok := args[0].(*object.String)
			if !ok {
				return NewError("argument 1 to `supabaseRpc` must be STRING")
			}

			return supabaseRequest("POST", "/rest/v1/rpc/"+functionName.Value, args[1], "")
		},
	}
}

func supabaseRequest(method, path string, payload object.Object, prefer string) object.Object {
	var bodyReader io.Reader

	if payload != nil {
		payloadBytes, err := json.Marshal(objectToGoValue(payload))
		if err != nil {
			return NewError("failed to encode supabase payload. got %s", err)
		}
		bodyReader = bytes.NewBuffer(payloadBytes)
	}

	req, err := http.NewRequest(method, supabaseURL+path, bodyReader)
	if err != nil {
		return NewError("failed to create supabase request. got %s", err)
	}

	req.Header.Set("apikey", supabaseKey)
	req.Header.Set("Authorization", "Bearer "+supabaseKey)
	req.Header.Set("Content-Type", "application/json")

	if prefer != "" {
		req.Header.Set("Prefer", prefer)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return NewError("failed to call supabase. got %s", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return NewError("supabase request failed. status=%d body=%s", resp.StatusCode, string(body))
	}

	if len(body) == 0 {
		return &object.Null{}
	}

	var decoded interface{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return &object.String{Value: string(body)}
	}

	return goValueToObject(decoded)
}

func SupabaseCountBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseCount(table, filterQuery). got=%d, want=2", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			table, ok1 := args[0].(*object.String)
			filterQuery, ok2 := args[1].(*object.String)
			if !ok1 || !ok2 {
				return NewError("supabaseCount(table, filterQuery) expects STRING, STRING")
			}

			qs := strings.TrimPrefix(filterQuery.Value, "?")
			if qs != "" {
				qs += "&"
			}
			qs += "select=*"

			req, err := http.NewRequest("GET", supabaseURL+"/rest/v1/"+table.Value+"?"+qs, nil)
			if err != nil {
				return NewError("failed to create supabase count request. got %s", err)
			}

			req.Header.Set("apikey", supabaseKey)
			req.Header.Set("Authorization", "Bearer "+supabaseKey)
			req.Header.Set("Prefer", "count=exact")
			req.Header.Set("Range", "0-0")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return NewError("failed to call supabase count. got %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				body, _ := io.ReadAll(resp.Body)
				return NewError("supabase count failed. status=%d body=%s", resp.StatusCode, string(body))
			}

			rangeHeader := resp.Header.Get("Content-Range")
			if rangeHeader == "" {
				return &object.Integer{Value: 0}
			}

			parts := strings.Split(rangeHeader, "/")
			if len(parts) != 2 {
				return &object.Integer{Value: 0}
			}

			total, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return &object.Integer{Value: 0}
			}

			return &object.Integer{Value: total}
		},
	}
}

func SupabaseSingleBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseSingle(table, queryString). got=%d, want=2", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			table, ok1 := args[0].(*object.String)
			queryString, ok2 := args[1].(*object.String)
			if !ok1 || !ok2 {
				return NewError("supabaseSingle(table, queryString) expects STRING, STRING")
			}

			qs := strings.TrimPrefix(queryString.Value, "?")
			req, err := http.NewRequest("GET", supabaseURL+"/rest/v1/"+table.Value+"?"+qs, nil)
			if err != nil {
				return NewError("failed to create supabase single request. got %s", err)
			}

			req.Header.Set("apikey", supabaseKey)
			req.Header.Set("Authorization", "Bearer "+supabaseKey)
			req.Header.Set("Accept", "application/vnd.pgrst.object+json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return NewError("failed to call supabase single. got %s", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode >= 400 {
				return NewError("supabase single failed. status=%d body=%s", resp.StatusCode, string(body))
			}

			var decoded interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				return &object.String{Value: string(body)}
			}

			return goValueToObject(decoded)
		},
	}
}

func SupabaseStorageUploadBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return NewError("wrong number of arguments, supabaseStorageUpload(bucket, path, localFilePath). got=%d, want=3", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			bucket, ok1 := args[0].(*object.String)
			remotePath, ok2 := args[1].(*object.String)
			localFilePath, ok3 := args[2].(*object.String)
			if !ok1 || !ok2 || !ok3 {
				return NewError("supabaseStorageUpload(bucket, path, localFilePath) expects STRING, STRING, STRING")
			}

			fileBytes, err := os.ReadFile(localFilePath.Value)
			if err != nil {
				return NewError("failed to read local file. got %s", err)
			}

			url := fmt.Sprintf("%s/storage/v1/object/%s/%s", supabaseURL, bucket.Value, strings.TrimPrefix(remotePath.Value, "/"))
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(fileBytes))
			if err != nil {
				return NewError("failed to create storage upload request. got %s", err)
			}

			req.Header.Set("apikey", supabaseKey)
			req.Header.Set("Authorization", "Bearer "+supabaseKey)
			req.Header.Set("Content-Type", "application/octet-stream")
			req.Header.Set("x-upsert", "true")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return NewError("failed to upload file to supabase storage. got %s", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode >= 400 {
				return NewError("supabase storage upload failed. status=%d body=%s", resp.StatusCode, string(body))
			}

			var decoded interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				return &object.String{Value: string(body)}
			}

			return goValueToObject(decoded)
		},
	}
}

func SupabaseStorageDeleteBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseStorageDelete(bucket, paths). got=%d, want=2", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			bucket, ok := args[0].(*object.String)
			if !ok {
				return NewError("argument 1 to `supabaseStorageDelete` must be STRING")
			}

			payloadBytes, err := json.Marshal(map[string]interface{}{
				"prefixes": objectToGoValue(args[1]),
			})
			if err != nil {
				return NewError("failed to encode storage delete payload. got %s", err)
			}

			url := fmt.Sprintf("%s/storage/v1/object/%s", supabaseURL, bucket.Value)
			req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(payloadBytes))
			if err != nil {
				return NewError("failed to create storage delete request. got %s", err)
			}

			req.Header.Set("apikey", supabaseKey)
			req.Header.Set("Authorization", "Bearer "+supabaseKey)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return NewError("failed to delete supabase storage object. got %s", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode >= 400 {
				return NewError("supabase storage delete failed. status=%d body=%s", resp.StatusCode, string(body))
			}

			var decoded interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				return &object.String{Value: string(body)}
			}

			return goValueToObject(decoded)
		},
	}
}

func SupabaseAuthSignUpBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseAuthSignUp(email, password). got=%d, want=2", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			email, ok1 := args[0].(*object.String)
			password, ok2 := args[1].(*object.String)
			if !ok1 || !ok2 {
				return NewError("supabaseAuthSignUp(email, password) expects STRING, STRING")
			}

			payloadBytes, _ := json.Marshal(map[string]interface{}{
				"email":    email.Value,
				"password": password.Value,
			})

			req, err := http.NewRequest("POST", supabaseURL+"/auth/v1/signup", bytes.NewBuffer(payloadBytes))
			if err != nil {
				return NewError("failed to create supabase signup request. got %s", err)
			}

			req.Header.Set("apikey", supabaseKey)
			req.Header.Set("Authorization", "Bearer "+supabaseKey)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return NewError("failed to call supabase signup. got %s", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode >= 400 {
				return NewError("supabase signup failed. status=%d body=%s", resp.StatusCode, string(body))
			}

			var decoded interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				return &object.String{Value: string(body)}
			}

			return goValueToObject(decoded)
		},
	}
}

func SupabaseAuthSignInBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseAuthSignIn(email, password). got=%d, want=2", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			email, ok1 := args[0].(*object.String)
			password, ok2 := args[1].(*object.String)
			if !ok1 || !ok2 {
				return NewError("supabaseAuthSignIn(email, password) expects STRING, STRING")
			}

			payloadBytes, _ := json.Marshal(map[string]interface{}{
				"email":    email.Value,
				"password": password.Value,
			})

			req, err := http.NewRequest("POST", supabaseURL+"/auth/v1/token?grant_type=password", bytes.NewBuffer(payloadBytes))
			if err != nil {
				return NewError("failed to create supabase signin request. got %s", err)
			}

			req.Header.Set("apikey", supabaseKey)
			req.Header.Set("Authorization", "Bearer "+supabaseKey)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return NewError("failed to call supabase signin. got %s", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode >= 400 {
				return NewError("supabase signin failed. status=%d body=%s", resp.StatusCode, string(body))
			}

			var decoded interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				return &object.String{Value: string(body)}
			}

			return goValueToObject(decoded)
		},
	}
}

func SupabaseStoragePublicUrlBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseStoragePublicUrl(bucket, path). got=%d, want=2", len(args))
			}
			if supabaseURL == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			bucket, ok1 := args[0].(*object.String)
			path, ok2 := args[1].(*object.String)
			if !ok1 || !ok2 {
				return NewError("supabaseStoragePublicUrl(bucket, path) expects STRING, STRING")
			}

			publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s",
				supabaseURL,
				bucket.Value,
				strings.TrimPrefix(path.Value, "/"),
			)

			return &object.String{Value: publicURL}
		},
	}
}

func SupabaseStorageSignedUrlBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return NewError("wrong number of arguments, supabaseStorageSignedUrl(bucket, path, expiresIn). got=%d, want=3", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			bucket, ok1 := args[0].(*object.String)
			path, ok2 := args[1].(*object.String)
			expiresIn, ok3 := args[2].(*object.Integer)
			if !ok1 || !ok2 || !ok3 {
				return NewError("supabaseStorageSignedUrl(bucket, path, expiresIn) expects STRING, STRING, INTEGER")
			}

			payloadBytes, _ := json.Marshal(map[string]interface{}{
				"expiresIn": expiresIn.Value,
			})

			url := fmt.Sprintf("%s/storage/v1/object/sign/%s/%s",
				supabaseURL,
				bucket.Value,
				strings.TrimPrefix(path.Value, "/"),
			)

			req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
			if err != nil {
				return NewError("failed to create signed url request. got %s", err)
			}

			req.Header.Set("apikey", supabaseKey)
			req.Header.Set("Authorization", "Bearer "+supabaseKey)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return NewError("failed to call supabase signed url. got %s", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode >= 400 {
				return NewError("supabase signed url failed. status=%d body=%s", resp.StatusCode, string(body))
			}

			var decoded interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				return &object.String{Value: string(body)}
			}

			return goValueToObject(decoded)
		},
	}
}

func SupabaseStorageDownloadBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseStorageDownload(bucket, path). got=%d, want=2", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			bucket, ok1 := args[0].(*object.String)
			path, ok2 := args[1].(*object.String)
			if !ok1 || !ok2 {
				return NewError("supabaseStorageDownload(bucket, path) expects STRING, STRING")
			}

			url := fmt.Sprintf("%s/storage/v1/object/%s/%s",
				supabaseURL,
				bucket.Value,
				strings.TrimPrefix(path.Value, "/"),
			)

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return NewError("failed to create storage download request. got %s", err)
			}

			req.Header.Set("apikey", supabaseKey)
			req.Header.Set("Authorization", "Bearer "+supabaseKey)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return NewError("failed to download supabase storage object. got %s", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode >= 400 {
				return NewError("supabase storage download failed. status=%d body=%s", resp.StatusCode, string(body))
			}

			return &object.String{Value: base64.StdEncoding.EncodeToString(body)}
		},
	}
}

func objectToPlainString(obj object.Object) string {
	switch v := obj.(type) {
	case *object.String:
		return v.Value
	default:
		return v.Inspect()
	}
}
