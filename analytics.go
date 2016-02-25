package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
)

type Analytic struct {
	CubeaconID   string `json:"cubeaconid"`
	Interactions int    `json:"interactions"`
	Created      int64  `json:"created"`
}

type AnalyticsResponse struct {
	Result       int    `json:"result"`
	Message      string `json:"message"`
	Interactions int64  `json:"interactions"`
}

type AnalyticsComparation struct {
	Label string `json:"label"`
	Data  int    `json:"data"`
}

func GetAnalytics(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	vars := mux.Vars(r)
	cubeaconid := vars["cubeaconid"]

	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB("evalboard").C("analytics")

	result := AnalyticsResponse{}
	err = c.Find(bson.M{"cubeaconid": cubeaconid}).One(&result)
	if err != nil {
		log.Println("error:", err)
		result.Result = 0
		result.Message = "Interaction is empty. Make sure data has sent successfully."
	} else {
		log.Println("success")
		result.Result = 1
		result.Message = "Interactions found"
	}

	b, err := json.Marshal(result)
	if err != nil {
		log.Println("error:", err)
	}

	w.Write([]byte(b))
}

func GetBeaconAnalytics(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")

	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB("evalboard").C("logs")

	pipeline := []bson.M{
		{"$match": bson.M{"flag": "ENTER"}},
		{"$group": bson.M{
			"_id": "$sensor",
			"data": bson.M{
				"$sum": 1,
			},
		},
		},
	}

	pipe := c.Pipe(pipeline)

	// Run the queries and capture the results
	results := []bson.M{}
	err1 := pipe.All(&results)

	if err1 != nil {
		fmt.Printf("ERROR : %s\n", err1.Error())
		return
	}

	b, err := json.Marshal(results)
	if err != nil {
		log.Println("error:", err)
	}
	fmt.Println(results)

	w.Write([]byte(b))

}

func GetFlagAnalytics(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")

	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB("evalboard").C("logs")

	// ENTER

	pipeline := []bson.M{
		{"$match": bson.M{"flag": "ENTER"}},
		{"$group": bson.M{
			"_id": bson.M{"flag": "$flag", "day": bson.M{"$dayOfYear": "$created"}, "year": bson.M{"$year": "$created"}},
			"data": bson.M{
				"$sum": 1,
			},
		},
		},
		{"$project": bson.M{"_id": 0, "label": "$_id.flag", "data": 1}},
		{"$sort": bson.M{"created": 1}},
		{"$limit": 10},
	}

	pipe := c.Pipe(pipeline)
	resultsEnter := []AnalyticsComparation{}
	iter := pipe.Iter()

	if err := iter.All(&resultsEnter); err != nil {
		fmt.Printf("error: %v", err.Error())
		return
	}

	// EXIT

	pipeline2 := []bson.M{
		{"$match": bson.M{"flag": "EXIT"}},
		{"$group": bson.M{
			"_id": bson.M{"flag": "$flag", "day": bson.M{"$dayOfYear": "$created"}, "year": bson.M{"$year": "$created"}},
			"data": bson.M{
				"$sum": 1,
			},
		},
		},
		{"$project": bson.M{"_id": 0, "label": "$_id.flag", "data": 1}},
		{"$sort": bson.M{"created": 1}},
		{"$limit": 10},
	}

	pipe2 := c.Pipe(pipeline2)

	resultsExit := []AnalyticsComparation{}
	iter2 := pipe2.Iter()

	if err := iter2.All(&resultsExit); err != nil {
		fmt.Printf("error: %v", err.Error())
		return
	}

	results := make(map[string]interface{})
	results["data1"] = resultsEnter
	results["data2"] = resultsExit

	b, err := json.Marshal(results)
	if err != nil {
		log.Println("error:", err)
	}

	w.Write([]byte(b))

}
