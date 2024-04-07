package cookie

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

const TOKENEXP = time.Hour * 3
const SECRETKEY = "supersecretkey"

func createJWTString(userID int) (tokenString string, err error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKENEXP)),
		},
		UserID: userID,
	})

	tokenString, err = token.SignedString([]byte(SECRETKEY))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func getUserID(tokenString string) (int, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(SECRETKEY), nil
		})

	if err != nil {
		return 0, err
	}

	if !token.Valid {
		return 0, ErrInvalidToken
	}

	return claims.UserID, nil
}

func CreateCookieClientID(userID int) *http.Cookie {
	jwtString, err := createJWTString(userID)
	if err != nil {
		log.Println(err)
	}
	cookie := &http.Cookie{
		Name:     "ClientID",
		Value:    jwtString,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   3600,
	}
	return cookie
}

func SetCookieMiddleware() func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			_, err := r.Cookie("ClientID")
			if err != nil {
				h.ServeHTTP(w, r)
				userID := w.Header().Get("ClientID")
				w.Header().Del("ClientID")
				userIDint, err := strconv.Atoi(userID)
				if err != nil {
					log.Println(err)
				}
				createdCookie := CreateCookieClientID(userIDint)
				http.SetCookie(w, createdCookie)
				w.WriteHeader(http.StatusOK)
			} else {
				h.ServeHTTP(w, r)
			}
		}
		return http.HandlerFunc(fn)
	}
}

func CheckCookieMiddleware() func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			reseivedCookie, err := r.Cookie("ClientID")
			if err != nil {
				switch {
				case errors.Is(err, http.ErrNoCookie):
					w.WriteHeader(http.StatusUnauthorized)
				default:
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}

			tokenString := reseivedCookie.Value

			if tokenString == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			userID, err := getUserID(tokenString)
			if err != nil {
				switch {
				case errors.Is(err, ErrInvalidToken):
					w.WriteHeader(http.StatusUnauthorized)
				default:
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}
			userIDstring := strconv.Itoa(userID)
			r.Header.Set("ClientID", userIDstring)
			h.ServeHTTP(w, r)

		}
		return http.HandlerFunc(fn)
	}
}
