package main 
import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"github.com/go-redis/redis/v8"		// работа с редисом
	"github.com/rs/cors"			    // работа с корсами что бы с разными доменами все работало
	"golang.org/x/crypto/acme/autocert"	// let's encrypt - налету генерить ssl сертификаты 
)

//установка соединения с Redis
var ctx = context.Background()
var rdb = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	Password: "", 						//нет пароля
	DB:       0, 						// дефолтная БД
})

// структура приходит на вход сервера что бы мы могли ее парсить
type analyticsData struct {
	HitType string `json: "hit_type"`
	PageType string `json: "page_type"`
	MaterialPK int `json: "material_pk"`
	EventCategory string `json: "event_category"`
	EventAction string `json: "event_action"`
	EventLabel int `json: "event_label"`
	EventValue int `json: "event_value"`
	Email string `json: "email"`
}

// структура для неуспешного ответа сервера 
type unseccessfulJSONResponse struct{
	Success bool `json: "success"`
	ErrorMessage string `json:"errorMessage"`
}

//структура успешного ответа сервера  
type seccessfulJSONResponse struct{
	Success bool `json: "success"`
	ErrorMessage string `json:"errorMessage"`
}

func Find(slice []string, val string)(int,bool){
	for i, item:= range slice{
		if item == val{
			return i, true
		}
	}
	return -1, false
}

func main(){
	mux := http.NewServeMux()
	corsMiddleware := cors.New(cors.Options{    // работа с корсами
		AllowedOrigins: []string{"http://istories.media"},	// домен с которого все должно работать
	})

	mux.HandleFunc("/send/", analyticsHandler)

	//Insert the middleware
	handler := corsMiddleware.Hadler(mux)
	//запускаем сервер и получаем ssl сертификат для домена и устанавливаем его (Let's encrypt механизмы)
	log.Fatal(http.Serve(autocert.NewListener("analytics.istories.media"), handler))
}


func ProcessSuccessfulDonate(MaterialPK int, email string){
	_,err := rdb.Append(ctx,
		fmt.Sprintf("donaters_of_material_%d", materialPK),
		fmt.Sprintf("%v", email),
	).Result()
	if err != nil{
		panic(err)
	}
}


func analyticsHandler(w http.ResponseWritter, r *http.Request){
	w.Header().Set("Content-Type", "application/json; charset= utf-8")

	decoder := json.NewDecoder(r.Body)
	var analyticsData analyticsData
	err := decoder.Decode(&analyticsData)
	if err != nil{
		writeUnsuccessfulResponse(w, "Can't parse JSON: " + err.Error())
		return
	}

	if analyticsData.HitType == "page_view"{
		ProcessMaterialView(analyticsData.MaterialPK)
		writeSuccessfulResponse(w, "")
	} else if analyticsData.HitType == "event"{
		if analyticsData.EventCategory != "donations"{
			writeUnsuccessfulResponse(w,"Unknown event_category")
			return
		}
		_, EventActionExists := Find(
			[]string{"submit", "success", "failure"},
			analyticsData.EventAction)
		if !EventActionExists{
			writeUnsuccessfulResponse(w,"Unknown event_action")
			return
		}
		if analyticsData.EventAction == "success"{
			ProcessSuccessfulDonate(analyticsData.EventLabel, analyticsData.Email)
		}
		writeSuccessfulResponse(w,"")
		return
	} else {
		writeUnsuccessfulResponse(w,"Unknown hit_type")
		return 
	}
}