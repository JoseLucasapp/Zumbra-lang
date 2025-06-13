package runtime

func Runtime() string {
	return `
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



`
}
