package main

		import (
			"sort"
			"fmt"
			"time"
			"bufio"
			"os"
			"strings"
			"crypto/sha256"
			"math"
			"math/rand"
			"encoding/json"
			"strconv"
			"errors"

			"github.com/golang-jwt/jwt/v5"
		)

		

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

	func dotenvGet(key string) string {
		return EnvVars[key]
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
		switch v := value.(type) {
		case string:
			return v != ""
		case float64:
			return v != 0
		case bool:
			return v
		case int:
			return v != 0
		default:
			return false
		}
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



		func main() {
			    fmt.Println(toUppercase("lucas"))
		}
	