package runtime

func Runtime() string {
	return `

func sizeOf(value interface{}) int {
	switch v := value.(type) {
	case []interface{}:
		return len(v)
	case string:
		return len(v)
	default:
		return 0
	}
}

func toUppercase(s string) string {
	return strings.ToUpper(s)
}

func toLowercase(s string) string {
	return strings.ToLower(s)
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func removeWhiteSpaces(s string) string {
	return strings.ReplaceAll(s, " ", "")
}

func replace(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

func addToArrayStart(arr []interface{}, elem interface{}) []interface{} {
	return append([]interface{}{elem}, arr...)
}

func addToArrayEnd(arr []interface{}, elem interface{}) []interface{} {
	return append(arr, elem)
}

func removeFromArray(arr []interface{}, index int) []interface{} {
	if index < 0 || index >= len(arr) {
		return arr
	}
	return append(arr[:index], arr[index+1:]...)
}

func max(arr []interface{}) interface{} {
	if len(arr) == 0 {
		return nil
	}
	maxVal := arr[0].(int)
	for _, v := range arr[1:] {
		val := v.(int)
		if val > maxVal {
			maxVal = val
		}
	}
	return maxVal
}

func min(arr []interface{}) interface{} {
	if len(arr) == 0 {
		return nil
	}
	minVal := arr[0].(int)
	for _, v := range arr[1:] {
		val := v.(int)
		if val < minVal {
			minVal = val
		}
	}
	return minVal
}

func first(arr []interface{}) interface{} {
	if len(arr) == 0 {
		return nil
	}
	return arr[0]
}

func last(arr []interface{}) interface{} {
	if len(arr) == 0 {
		return nil
	}
	return arr[len(arr)-1]
}

func allButFirst(arr []interface{}) []interface{} {
	if len(arr) == 0 {
		return arr
	}
	return arr[1:]
}

func indexOf(arr []interface{}, elem interface{}) int {
	for i, v := range arr {
		if v == elem {
			return i
		}
	}
	return -1
}

func organize(arr []interface{}, order string) []interface{} {
	intArr := make([]int, len(arr))
	for i, v := range arr {
		intArr[i] = v.(int)
	}
	if order == "desc" {
		sort.Sort(sort.Reverse(sort.IntSlice(intArr)))
	} else {
		sort.Ints(intArr)
	}
	result := make([]interface{}, len(intArr))
	for i, v := range intArr {
		result[i] = v
	}
	return result
}

func sum(arr []interface{}) interface{} {
	total := 0.0
	for _, v := range arr {
		switch val := v.(type) {
		case int:
			total += float64(val)
		case float64:
			total += val
		}
	}
	if float64(int(total)) == total {
		return int(total)
	}
	return total
}

type ZumbraDate struct {
	fullDate time.Time
	hour     int
	minute   int
	second   int
	day      int
	month    int
	year     int
}

func date() ZumbraDate {
	now := time.Now()
	return ZumbraDate{
		fullDate: now,
		hour:     now.Hour(),
		minute:   now.Minute(),
		second:   now.Second(),
		day:      now.Day(),
		month:    int(now.Month()),
		year:     now.Year(),
	}
}

func addToDict(dict map[string]interface{}, key string, value interface{}) map[string]interface{} {
	dict[key] = value
	return dict
}

func deleteFromDict(dict map[string]interface{}, key string) map[string]interface{} {
	delete(dict, key)
	return dict
}

func getFromDict(dict map[string]interface{}, key string) interface{} {
	return dict[key]
}

func dictKeys(dict map[string]interface{}) []string {
	keys := make([]string, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	return keys
}

func dictValues(dict map[string]interface{}) []interface{} {
	values := make([]interface{}, 0, len(dict))
	for _, v := range dict {
		values = append(values, v)
	}
	return values
}

var EnvVars = map[string]string{}

func dotenvLoad(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println("failed to open file:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			EnvVars[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("failed to read file:", err)
	}
}

func dotenvGet(key string) interface{} {
	if value, ok := EnvVars[key]; ok {
		return value
	}
	return nil
}

func hashCode(input string) string {
	hash := sha256.New()
	hash.Write([]byte(input))
	hashInBytes := hash.Sum(nil)
	return fmt.Sprintf("%x", hashInBytes)
}

func input(prompt ...string) string {
	if len(prompt) > 0 {
		fmt.Print(prompt[0])
	}
	var value string
	fmt.Scanln(&value)
	return value
}

func bhaskara(a, b, c float64) interface{} {
	delta := (b * b) - (4 * a * c)
	if delta < 0 {
		return nil
	}
	if delta == 0 {
		return -b / (2 * a)
	}
	sqrtDelta := math.Sqrt(delta)
	x1 := (-b + sqrtDelta) / (2 * a)
	x2 := (-b - sqrtDelta) / (2 * a)
	return []interface{}{x1, x2}
}

func randomInteger(args ...int) int {
	min := 0
	max := 10
	if len(args) == 1 {
		max = args[0]
	} else if len(args) == 2 {
		min = args[0]
		max = args[1]
	}
	if min > max {
		min, max = max, min
	}
	return min + rand.Intn(max-min+1)
}

func randomFloat(args ...float64) float64 {
	min := 0.0
	max := 10.0
	if len(args) == 1 {
		max = args[0]
	} else if len(args) == 2 {
		min = args[0]
		max = args[1]
	}
	if min > max {
		min, max = max, min
	}
	return min + rand.Float64()*(max-min)
}

func toString(value interface{}) string {
	return fmt.Sprintf("%v", value)
}

func toInt(value interface{}) int {
	switch v := value.(type) {
	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0
		}
		return n
	case float64:
		return int(math.Floor(v))
	case bool:
		if v {
			return 1
		}
		return 0
	case int:
		return v
	default:
		return 0
	}
}

func toFloat(value interface{}) float64 {
	switch v := value.(type) {
	case string:
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0
		}
		return n
	case float64:
		return v
	case bool:
		if v {
			return 1.0
		}
		return 0.0
	case int:
		return float64(v)
	default:
		return 0.0
	}
}

func toBool(value interface{}) bool {
	return isTruthy(value)
}


func isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case string:
		return v != ""
	case bool:
		return v
	case int:
		return v != 0
	case int8:
		return v != 0
	case int16:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case float32:
		return v != 0
	case float64:
		return v != 0
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		return true
	}
}

func zOr(left interface{}, right interface{}) interface{} {
	if isTruthy(left) {
		return left
	}
	return right
}

func zAnd(left interface{}, right interface{}) interface{} {
	if isTruthy(left) {
		return right
	}
	return left
}

func jsonParse(input string) map[string]interface{} {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(input), &result)
	if err != nil {
		return map[string]interface{}{}
	}
	return result
}

var secretKey string

func jwtCreateToken(username string, secret string, expirationHours int) (string, error) {
	secretKey = secret

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Hour * time.Duration(expirationHours)).Unix(),
	})

	tokenStr, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to create token: %v", err)
	}

	return tokenStr, nil
}

func jwtVerifyToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse token: %v", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		username, ok := claims["username"].(string)
		if !ok {
			return "", errors.New("username not found in token")
		}
		return username, nil
	}

	return "", errors.New("invalid token")
}

var db_connection *sql.DB

func mysqlConnection(host, port, user, password, database string) error {
	var err error
	db_connection, err = sql.Open("mysql", user+":"+password+"@tcp("+host+":"+port+")/"+database)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	err = db_connection.Ping()
	if err != nil {
		return fmt.Errorf("failed to ping: %v", err)
	}

	fmt.Printf("Database '%s' connected successfully\n", database)
	return nil
}

func mysqlCreateTable(tableName, fields string) error {
	_, err := db_connection.Exec("CREATE TABLE " + tableName + " (" + fields + ");")
	if err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}
	fmt.Printf("Table '%s' created successfully\n", tableName)
	return nil
}

func mysqlShowTables() ([]string, error) {
	rows, err := db_connection.Query("SHOW TABLES")
	if err != nil {
		return nil, fmt.Errorf("failed to show tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, nil
}

func mysqlShowTableColumns(tableName string) ([]string, error) {
	rows, err := db_connection.Query("SHOW COLUMNS FROM " + tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to show columns: %v", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var field, colType, null, key, extra string
		var defaultValue sql.NullString
		if err := rows.Scan(&field, &colType, &null, &key, &defaultValue, &extra); err != nil {
			return nil, err
		}
		columns = append(columns, field)
	}
	return columns, nil
}

func mysqlDeleteTable(tableName string) error {
	query := fmt.Sprintf("DROP TABLE %s", tableName)
	_, err := db_connection.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to drop table: %v", err)
	}
	fmt.Printf("Table '%s' deleted successfully\n", tableName)
	return nil
}

func mysqlInsertIntoTable(tableName string, data map[string]interface{}) error {
	keys := []string{}
	placeholders := []string{}
	args := []interface{}{}

	for key, value := range data {
		keys = append(keys, key)
		placeholders = append(placeholders, "?")
		args = append(args, value)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableName, strings.Join(keys, ","), strings.Join(placeholders, ","))
	_, err := db_connection.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to insert: %v", err)
	}

	fmt.Println("Record inserted successfully")
	return nil
}

func mysqlGetFromTable(tableName, fields, condition string) ([]map[string]interface{}, error) {
	if db_connection == nil {
		return nil, errors.New("not connected")
	}

	query := fmt.Sprintf("SELECT %s FROM %s", fields, tableName)
	if condition != "" {
		query += " WHERE " + condition
	}

	rows, err := db_connection.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v", err)
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	result := []map[string]interface{}{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		rowMap := map[string]interface{}{}
		for i, col := range columns {
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		result = append(result, rowMap)
	}

	return result, nil
}

func mysqlUpdateIntoTable(tableName string, data map[string]interface{}, condition string) error {
	assignments := []string{}
	args := []interface{}{}

	for key, value := range data {
		assignments = append(assignments, fmt.Sprintf("%s = ?", key))
		args = append(args, value)
	}

	query := fmt.Sprintf("UPDATE %s SET %s", tableName, strings.Join(assignments, ", "))
	if condition != "" {
		query += " WHERE " + condition
	}

	_, err := db_connection.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update: %v", err)
	}
	fmt.Println("Record updated successfully")
	return nil
}

func mysqlDeleteFromTable(tableName, condition string) error {
	query := fmt.Sprintf("DELETE FROM %s", tableName)
	if condition != "" {
		query += " WHERE " + condition
	}
	_, err := db_connection.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to delete: %v", err)
	}
	fmt.Println("Record deleted successfully")
	return nil
}

func objectToPlainString(value interface{}) string {
	return fmt.Sprintf("%v", value)
}

func toStringSlice(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprintf("%v", item))
		}
		return out
	default:
		return []string{}
	}
}

func decodeJSONBody(body []byte) interface{} {
	if len(body) == 0 {
		return nil
	}

	var decoded interface{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return string(body)
	}
	return decoded
}

var postgresDB *sql.DB
var redisClient *redis.Client
var supabaseURL string
var supabaseKey string

func postgresConnection(connectionString string) error {
	var err error
	postgresDB, err = sql.Open("postgres", connectionString)
	if err != nil {
		return fmt.Errorf("failed to open postgres connection: %v", err)
	}

	if err := postgresDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping postgres: %v", err)
	}

	return nil
}

func postgresExec(query string) error {
	if postgresDB == nil {
		return errors.New("postgres is not connected")
	}

	_, err := postgresDB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to exec postgres query: %v", err)
	}

	return nil
}

func postgresQuery(query string) ([]map[string]interface{}, error) {
	if postgresDB == nil {
		return nil, errors.New("postgres is not connected")
	}

	rows, err := postgresDB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query postgres: %v", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get postgres columns: %v", err)
	}

	result := []map[string]interface{}{}

	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("failed to scan postgres row: %v", err)
		}

		rowMap := map[string]interface{}{}
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}

		result = append(result, rowMap)
	}

	return result, nil
}

func redisConnection(addr, password string, db int) error {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis: %v", err)
	}

	return nil
}

func redisSet(key string, value interface{}) error {
	if redisClient == nil {
		return errors.New("redis is not connected")
	}

	ctx := context.Background()
	if err := redisClient.Set(ctx, key, objectToPlainString(value), 0).Err(); err != nil {
		return fmt.Errorf("failed to set redis key: %v", err)
	}

	return nil
}

func redisGet(key string) interface{} {
	if redisClient == nil {
		return nil
	}

	ctx := context.Background()
	value, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return nil
	}

	return value
}

func redisDel(key string) error {
	if redisClient == nil {
		return errors.New("redis is not connected")
	}

	ctx := context.Background()
	if err := redisClient.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete redis key: %v", err)
	}

	return nil
}

func supabaseConnection(url, key string) {
	supabaseURL = strings.TrimRight(url, "/")
	supabaseKey = key
}

func supabaseRequest(method, path string, payload interface{}, prefer string, extraHeaders map[string]string) interface{} {
	var bodyReader io.Reader

	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil
		}
		bodyReader = bytes.NewBuffer(payloadBytes)
	}

	req, err := http.NewRequest(method, supabaseURL+path, bodyReader)
	if err != nil {
		return nil
	}

	req.Header.Set("apikey", supabaseKey)
	req.Header.Set("Authorization", "Bearer "+supabaseKey)
	req.Header.Set("Content-Type", "application/json")

	if prefer != "" {
		req.Header.Set("Prefer", prefer)
	}

	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return map[string]interface{}{
			"error":  true,
			"status": resp.StatusCode,
			"body":   string(body),
		}
	}

	return decodeJSONBody(body)
}

func supabaseSelect(table, selectQuery string) interface{} {
	query := "select=" + neturl.QueryEscape(selectQuery)
	return supabaseRequest("GET", "/rest/v1/"+table+"?"+query, nil, "", nil)
}

func supabaseQuery(table, queryString string) interface{} {
	qs := strings.TrimPrefix(queryString, "?")
	return supabaseRequest("GET", "/rest/v1/"+table+"?"+qs, nil, "", nil)
}

func supabaseSingle(table, queryString string) interface{} {
	qs := strings.TrimPrefix(queryString, "?")
	return supabaseRequest("GET", "/rest/v1/"+table+"?"+qs, nil, "", map[string]string{
		"Accept": "application/vnd.pgrst.object+json",
	})
}

func supabaseCount(table, filterQuery string) int {
	qs := strings.TrimPrefix(filterQuery, "?")
	if qs != "" {
		qs += "&"
	}
	qs += "select=*"

	req, err := http.NewRequest("GET", supabaseURL+"/rest/v1/"+table+"?"+qs, nil)
	if err != nil {
		return 0
	}

	req.Header.Set("apikey", supabaseKey)
	req.Header.Set("Authorization", "Bearer "+supabaseKey)
	req.Header.Set("Prefer", "count=exact")
	req.Header.Set("Range", "0-0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	rangeHeader := resp.Header.Get("Content-Range")
	if rangeHeader == "" {
		return 0
	}

	parts := strings.Split(rangeHeader, "/")
	if len(parts) != 2 {
		return 0
	}

	total, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}

	return total
}

func supabaseInsert(table string, payload interface{}) interface{} {
	return supabaseRequest("POST", "/rest/v1/"+table, payload, "return=representation", nil)
}

func supabaseUpdate(table, filterQuery string, payload interface{}) interface{} {
	qs := strings.TrimPrefix(filterQuery, "?")
	path := "/rest/v1/" + table
	if qs != "" {
		path += "?" + qs
	}
	return supabaseRequest("PATCH", path, payload, "return=representation", nil)
}

func supabaseDelete(table, filterQuery string) interface{} {
	qs := strings.TrimPrefix(filterQuery, "?")
	path := "/rest/v1/" + table
	if qs != "" {
		path += "?" + qs
	}
	return supabaseRequest("DELETE", path, nil, "return=representation", nil)
}

func supabaseUpsert(table string, payload interface{}) interface{} {
	return supabaseRequest("POST", "/rest/v1/"+table, payload, "resolution=merge-duplicates,return=representation", nil)
}

func supabaseRpc(functionName string, payload interface{}) interface{} {
	return supabaseRequest("POST", "/rest/v1/rpc/"+functionName, payload, "", nil)
}

func supabaseStorageUpload(bucket, remotePath, localFilePath string) interface{} {
	fileBytes, err := os.ReadFile(localFilePath)
	if err != nil {
		return map[string]interface{}{"error": true, "body": err.Error()}
	}

	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", supabaseURL, bucket, strings.TrimPrefix(remotePath, "/"))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(fileBytes))
	if err != nil {
		return map[string]interface{}{"error": true, "body": err.Error()}
	}

	req.Header.Set("apikey", supabaseKey)
	req.Header.Set("Authorization", "Bearer "+supabaseKey)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("x-upsert", "true")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return map[string]interface{}{"error": true, "body": err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return map[string]interface{}{
			"error":  true,
			"status": resp.StatusCode,
			"body":   string(body),
		}
	}

	return decodeJSONBody(body)
}

func supabaseStorageDelete(bucket string, paths interface{}) interface{} {
	payload := map[string]interface{}{
		"prefixes": toStringSlice(paths),
	}

	url := fmt.Sprintf("%s/storage/v1/object/%s", supabaseURL, bucket)
	reqBody, _ := json.Marshal(payload)

	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil
	}

	req.Header.Set("apikey", supabaseKey)
	req.Header.Set("Authorization", "Bearer "+supabaseKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return decodeJSONBody(body)
}

func supabaseStoragePublicUrl(bucket, path string) string {
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", supabaseURL, bucket, strings.TrimPrefix(path, "/"))
}

func supabaseStorageSignedUrl(bucket, path string, expiresIn int) interface{} {
	payload := map[string]interface{}{
		"expiresIn": expiresIn,
	}

	url := fmt.Sprintf("%s/storage/v1/object/sign/%s/%s", supabaseURL, bucket, strings.TrimPrefix(path, "/"))
	reqBody, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil
	}

	req.Header.Set("apikey", supabaseKey)
	req.Header.Set("Authorization", "Bearer "+supabaseKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return decodeJSONBody(body)
}

func supabaseStorageDownload(bucket, path string) string {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", supabaseURL, bucket, strings.TrimPrefix(path, "/"))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("apikey", supabaseKey)
	req.Header.Set("Authorization", "Bearer "+supabaseKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return ""
	}

	return base64.StdEncoding.EncodeToString(body)
}

func supabaseAuthSignUp(email, password string) interface{} {
	payload := map[string]interface{}{
		"email":    email,
		"password": password,
	}
	return supabaseRequest("POST", "/auth/v1/signup", payload, "", nil)
}

func supabaseAuthSignIn(email, password string) interface{} {
	payload := map[string]interface{}{
		"email":    email,
		"password": password,
	}
	return supabaseRequest("POST", "/auth/v1/token?grant_type=password", payload, "", nil)
}

func switchCase(key interface{}, cases map[string]interface{}, defaultValue interface{}) interface{} {
	strKey := fmt.Sprintf("%v", key)
	if value, ok := cases[strKey]; ok {
		return value
	}
	return defaultValue
}

func createFile(path string, content string) string {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return ""
	}
	return path
}

func createTxt(path string, content string) string {
	return createFile(path, content)
}

func createCsv(path string, content string) string {
	return createFile(path, content)
}

func createDoc(path string, title string, content string) string {
	doc := fmt.Sprintf("<html><head><meta charset=\"utf-8\"><title>%s</title></head><body><h1>%s</h1><p>%s</p></body></html>", title, title, strings.ReplaceAll(content, "\n", "<br>"))
	return createFile(path, doc)
}

func createPdf(path string, title string, content string) string {
	pdfText := "TITLE: " + title + "\n\n" + content
	return createFile(path, pdfText)
}

type ZRequest struct {
	Method  string
	Path    string
	Params  map[string]interface{}
	Query   map[string]interface{}
	Headers map[string]interface{}
	Body    interface{}
	RawBody string
}

type ZResponse struct {
	StatusCode  int
	Headers     map[string]string
	Body        interface{}
	ContentType string
}

type ZRoute struct {
	Method       string
	Path         string
	Handler      func(*ZRequest, *ZResponse)
	AsyncHandler ZAsyncHandler
	IsAsync      bool
}

type StaticRoute struct {
	Prefix string
	Dir    string
}

var routes []ZRoute
var staticRoutes []StaticRoute

func newResponse() *ZResponse {
	return &ZResponse{
		StatusCode: 200,
		Headers:    map[string]string{},
	}
}

func responseStatus(res *ZResponse, code int) *ZResponse {
	res.StatusCode = code
	return res
}

func responseHeader(res *ZResponse, key, value string) *ZResponse {
	res.Headers[key] = value
	return res
}

func responseJSON(res *ZResponse, data interface{}) *ZResponse {
	res.ContentType = "application/json"
	res.Body = data
	return res
}

func responseSend(res *ZResponse, data interface{}) *ZResponse {
	res.Body = data
	return res
}

func responseHTML(res *ZResponse, content string) *ZResponse {
	res.ContentType = "text/html; charset=utf-8"
	res.Body = content
	return res
}

func html(content string) string {
	return content
}

func restGet(path string, handler func(*ZRequest, *ZResponse)) {
	routes = append(routes, ZRoute{
		Method:  "GET",
		Path:    path,
		Handler: handler,
	})
}

func restPost(path string, handler func(*ZRequest, *ZResponse)) {
	routes = append(routes, ZRoute{
		Method:  "POST",
		Path:    path,
		Handler: handler,
	})
}

func restPut(path string, handler func(*ZRequest, *ZResponse)) {
	routes = append(routes, ZRoute{
		Method:  "PUT",
		Path:    path,
		Handler: handler,
	})
}

func restDelete(path string, handler func(*ZRequest, *ZResponse)) {
	routes = append(routes, ZRoute{
		Method:  "DELETE",
		Path:    path,
		Handler: handler,
	})
}

func restPatch(path string, handler func(*ZRequest, *ZResponse)) {
	routes = append(routes, ZRoute{
		Method:  "PATCH",
		Path:    path,
		Handler: handler,
	})
}

func restGetAsync(path string, handler ZAsyncHandler) {
	routes = append(routes, ZRoute{
		Method:       "GET",
		Path:         path,
		AsyncHandler: handler,
		IsAsync:      true,
	})
}

func restPostAsync(path string, handler ZAsyncHandler) {
	routes = append(routes, ZRoute{
		Method:       "POST",
		Path:         path,
		AsyncHandler: handler,
		IsAsync:      true,
	})
}

func restPutAsync(path string, handler ZAsyncHandler) {
	routes = append(routes, ZRoute{
		Method:       "PUT",
		Path:         path,
		AsyncHandler: handler,
		IsAsync:      true,
	})
}

func restDeleteAsync(path string, handler ZAsyncHandler) {
	routes = append(routes, ZRoute{
		Method:       "DELETE",
		Path:         path,
		AsyncHandler: handler,
		IsAsync:      true,
	})
}

func restPatchAsync(path string, handler ZAsyncHandler) {
	routes = append(routes, ZRoute{
		Method:       "PATCH",
		Path:         path,
		AsyncHandler: handler,
		IsAsync:      true,
	})
}

func registerRoute(path string, handler string) {
	restGet(path, func(req *ZRequest, res *ZResponse) {
		responseSend(res, handler)
	})
}

func serveStatic(prefix, dir string) {
	staticRoutes = append(staticRoutes, StaticRoute{
		Prefix: prefix,
		Dir:    dir,
	})
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

func matchRoute(method, path string) (*ZRoute, map[string]interface{}) {
	reqParts := splitPath(path)

	for _, route := range routes {
		if route.Method != method {
			continue
		}

		routeParts := splitPath(route.Path)
		if len(routeParts) != len(reqParts) {
			continue
		}

		params := map[string]interface{}{}
		matched := true

		for i := 0; i < len(routeParts); i++ {
			if strings.HasPrefix(routeParts[i], ":") {
				params[strings.TrimPrefix(routeParts[i], ":")] = reqParts[i]
				continue
			}

			if routeParts[i] != reqParts[i] {
				matched = false
				break
			}
		}

		if matched {
			routeCopy := route
			return &routeCopy, params
		}
	}

	return nil, nil
}

func normalizeQuery(values map[string][]string) map[string]interface{} {
	result := map[string]interface{}{}
	for key, arr := range values {
		if len(arr) == 1 {
			result[key] = arr[0]
		} else {
			tmp := make([]interface{}, 0, len(arr))
			for _, item := range arr {
				tmp = append(tmp, item)
			}
			result[key] = tmp
		}
	}
	return result
}

func normalizeHeaders(values map[string][]string) map[string]interface{} {
	result := map[string]interface{}{}
	for key, arr := range values {
		if len(arr) == 1 {
			result[strings.ToLower(key)] = arr[0]
		} else {
			tmp := make([]interface{}, 0, len(arr))
			for _, item := range arr {
				tmp = append(tmp, item)
			}
			result[strings.ToLower(key)] = tmp
		}
	}
	return result
}

func parseRequestBody(r *http.Request) (interface{}, string) {
	bodyBytes, _ := io.ReadAll(r.Body)
	raw := string(bodyBytes)

	if len(bodyBytes) == 0 {
		return nil, raw
	}

	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(contentType, "application/json") {
		var decoded interface{}
		if err := json.Unmarshal(bodyBytes, &decoded); err == nil {
			return decoded, raw
		}
	}

	return raw, raw
}

func writeResponse(w http.ResponseWriter, res *ZResponse) {
	if res == nil {
		w.WriteHeader(200)
		return
	}

	for k, v := range res.Headers {
		w.Header().Set(k, v)
	}

	if res.StatusCode <= 0 {
		res.StatusCode = 200
	}

	if res.Body == nil {
		w.WriteHeader(res.StatusCode)
		return
	}

	if res.ContentType != "" {
		w.Header().Set("Content-Type", res.ContentType)
	}

	switch body := res.Body.(type) {
	case string:
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}
		w.WriteHeader(res.StatusCode)
		_, _ = w.Write([]byte(body))
	case []byte:
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		w.WriteHeader(res.StatusCode)
		_, _ = w.Write(body)
	default:
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(res.StatusCode)
		encoded, err := json.Marshal(body)
		if err != nil {
	_, _ = w.Write([]byte("{\"error\":true,\"body\":\"failed to encode response\"}"))
	return
}
		_, _ = w.Write(encoded)
	}
}

func server(port int) {
	mux := http.NewServeMux()

	for _, sr := range staticRoutes {
		mux.Handle(sr.Prefix+"/", http.StripPrefix(sr.Prefix, http.FileServer(http.Dir(sr.Dir))))
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		route, params := matchRoute(r.Method, r.URL.Path)
		if route == nil {
			http.NotFound(w, r)
			return
		}

		body, rawBody := parseRequestBody(r)

		reqObj := &ZRequest{
			Method:  r.Method,
			Path:    r.URL.Path,
			Params:  params,
			Query:   normalizeQuery(r.URL.Query()),
			Headers: normalizeHeaders(r.Header),
			Body:    body,
			RawBody: rawBody,
		}

		resObj := newResponse()

if route.IsAsync && route.AsyncHandler != nil {
	result := route.AsyncHandler(reqObj, resObj)

	if isErrorValue(result) {
		if resObj.StatusCode == 200 {
			resObj.StatusCode = 500
		}

		if resObj.Body == nil {
			resObj.ContentType = "application/json"
			resObj.Body = map[string]interface{}{
				"error": true,
				"body":  fmt.Sprintf("%v", result),
			}
		}
	}
} else if route.Handler != nil {
	route.Handler(reqObj, resObj)
}

writeResponse(w, resObj)
	})

	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("Failed to bind to port %s. got %s\n", addr, err)
		return
	}

	fmt.Printf("Zumbra server started on port %d\n", port)
	_ = http.Serve(ln, mux)
}

type ZAsyncHandler func(*ZRequest, *ZResponse) interface{}

type ZTask struct {
	Value interface{}
	Err   interface{}
	Done  bool
}

func isErrorValue(v interface{}) bool {
	if v == nil {
		return false
	}

	switch val := v.(type) {
	case error:
		return true
	case map[string]interface{}:
		if e, ok := val["error"].(bool); ok && e {
			return true
		}
		return false
	case *ZTask:
		return val.Err != nil
	default:
		return false
	}
}

func errorValue(message string) interface{} {
	return map[string]interface{}{
		"error":   true,
		"message": message,
	}
}

func panicValue(message string) interface{} {
	panic(message)
}

func awaitValue(v interface{}) interface{} {
	switch t := v.(type) {
	case *ZTask:
		if t.Err != nil {
			return t.Err
		}
		return t.Value
	default:
		return v
	}
}

func tryValue(v interface{}) interface{} {
	return awaitValue(v)
}

`
}
