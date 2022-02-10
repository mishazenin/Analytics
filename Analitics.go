package main 
import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"github.com/go-redis/redis/v8"		
	"github.com/rs/cors"			   
	"golang.org/x/crypto/acme/autocert"	
)


var ctx = context.Background()
var rdb = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	Password: "", 						
	DB:       0, 						
})


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


type unseccessfulJSONResponse struct{
	Success bool `json: "success"`
	ErrorMessage string `json:"errorMessage"`
}


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



func writeSuccessfulResponse(w http.ResponseWriter, message string){
	w.WriteHeader(http.StatusOk)
	if message == ""{       						
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
	corsMiddleware := cors.New(cors.Options{    
		AllowedOrigins: []string{"http://istories.media"},	
	})

	mux.HandleFunc("/send/", analyticsHandler)

	
	handler := corsMiddleware.Hadler(mux)
	
	log.Fatal(http.Serve(autocert.NewListener("analytics.istories.media"), handler))
}


func ProcessMaterialView(materialPK int){
	_, err := edb.Incr(ctx, fmt.Sprintf("material_views_%d", materialPK)).Result()  
	if err != nil{
		panic(err)
	}
}

func ProcessSuccessfulDonate(MaterialPK int, email string){
	_,err := rdb.Append(ctx,
		fmt.Sprintf("donaters_of_material_%d", materialPK),   
		fmt.Sprintf("%v:", email),								
	).Result()
	if err != nil{
		panic(err)
	}
}


func analyticsHandler(w http.ResponseWriter, r *http.Request){
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
		_, eventActionExists := Find(								
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
