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

func writeUnsuccessfulResponse( w http.ResponseWriter, ErrorMessage string){
	response, _ := json.Marshal(&unseccessfulJSONResponse{
		Success: false,
		ErrorMessage: errMessage,
	})
	w.WriteHeader(http.StatusBadRequest)
	w.Write(response)
}


//ответ об успехе в браузер 
func writeSuccessfulResponse(w http.ResponseWriter, message string){
	w.WriteHeader(http.StatusOk)
	if message == ""{       						// если сообщение не передано то пишем...
		w.Write([]byte(`{"success": true}`))
	} else{
		response, _ := json.Marshal(&successfulJSONResponse{
			Success: true,
			Message: message,
		})
		w.Write(response)
	}
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

// просмотр страницы
func ProcessMaterialView(materialPK int){
	_, err := edb.Incr(ctx, fmt.Sprintf("material_views_%d", materialPK)).Result()  // Incr - увеличивает счетчие Redis, materialPk
	if err != nil{
		panic(err)
	}
}

func ProcessSuccessfulDonate(MaterialPK int, email string){
	_,err := rdb.Append(ctx,
		fmt.Sprintf("donaters_of_material_%d", materialPK),   // ключ в конце айди пользователя оформ донат
		fmt.Sprintf("%v:", email),								// мыло донатера
	).Result()
	if err != nil{
		panic(err)
	}
}


func analyticsHandler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json; charset= utf-8")   // уст контент тайп для ответа

	decoder := json.NewDecoder(r.Body) 										 // прост созд декодер 
	var analyticsData analyticsData
	err := decoder.Decode(&analyticsData)  									// тут декодим данные
	if err != nil{
		writeUnsuccessfulResponse(w, "Can't parse JSON: " + err.Error())  // сорян джейсон корявый пришел
		return
	}
	// принятие аналитических данных и сохранение их в Redis
	if analyticsData.HitType == "page_view"{ 						// чекаем тип сообщения, просмотр страницы
		ProcessMaterialView(analyticsData.MaterialPK)
		writeSuccessfulResponse(w, "")
	} else if analyticsData.HitType == "event"{						// событие ( успешная неуспешная отправка доната)
		if analyticsData.EventCategory != "donations"{				// разбираем структуру
			writeUnsuccessfulResponse(w,"Unknown event_category")
			return
		}
		_, eventActionExists := Find(								// ищем в структуре необхоимые ключи
			[]string{"submit", "success", "failure"},
			analyticsData.EventAction)
		if !eventActionExists{
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