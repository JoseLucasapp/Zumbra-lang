package builtins

import (
	"time"
	"zumbra/object"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey string

func createTokenBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return NewError("wrong number of arguments. got=%d, want=3", len(args))
			}

			if args[0].Type() != object.STRING_OBJ {
				return NewError("First argument to `createToken` must be STRING, got %s", args[0].Type())
			}

			if args[1].Type() != object.STRING_OBJ {
				return NewError("Secret key to `createToken` must be STRING, got %s", args[1].Type())
			}

			if args[2].Type() != object.INTEGER_OBJ {
				return NewError("Expiration to `createToken` must be INTEGER, it will be the expiration in hours, got %s", args[2].Type())
			}

			username := args[0].(*object.String).Value
			secretKey = args[1].(*object.String).Value
			expiration := args[2].(*object.Integer).Value

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"username": username,
				"exp":      time.Now().Add(time.Hour * time.Duration(expiration)).Unix(),
			})

			tokenStr, err := token.SignedString([]byte(secretKey))
			if err != nil {
				return NewError("Failed to create token, createToken('%s', '%s', '%d'). got %s", username, secretKey, expiration, err)
			}

			return &object.String{Value: tokenStr}
		},
	}
}

func verifyTokenBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			if args[0].Type() != object.STRING_OBJ {
				return NewError("Argument to `verifyToken` must be STRING, got %s", args[0].Type())
			}

			tokenStr := args[0].(*object.String).Value

			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					NewError("unexpected signing method: %v", token.Header["alg"])
					return nil, nil
				}
				return []byte(secretKey), nil
			})
			if err != nil {
				return NewError("Failed to verify token, verifyToken('%s'). got %s", tokenStr, err)
			}

			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				return &object.String{Value: claims["username"].(string)}
			}

			return NewError("Failed to verify token, verifyToken('%s'). got %s", tokenStr, err)
		},
	}
}
