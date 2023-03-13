package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"HlsToLiveForFFMPEG/server/service"

	"github.com/gorilla/mux"
)

type Response struct {
	w      http.ResponseWriter
	Status int         `json:"-"`
	Data   interface{} `json:"-"`
}

type Result struct {
	Result int         `json:"result"`
	Data   interface{} `json:"data"`
	Code   string      `json:"code"`
}

func (r *Response) SendJson(result int, code string) (int, error) {
	ret := Result{
		Result: result,
		Code:   code,
	}
	if result == 0 {
		ret.Data = struct{}{}
	} else {
		ret.Data = r.Data
		ret.Code = "0000"
	}

	resp, err := json.Marshal(ret)
	if err != nil {
		log.Println("[error]", "json Marshal :", err)
		ret.Data = struct{}{}
		ret.Code = "0002"
	}

	r.w.Header().Set("Content-Type", "application/json")
	r.w.WriteHeader(r.Status)
	return r.w.Write(resp)
}

func (s *Server) Serve() {
	//mux := http.NewServeMux()
	mu := mux.NewRouter()
	mu.HandleFunc("/SetList", func(w http.ResponseWriter, r *http.Request) {
		s.handleSetList(w, r)
	})
	mu.HandleFunc("/all", func(w http.ResponseWriter, r *http.Request) {
		s.handleAll(w, r)
	})
	mu.HandleFunc("/remove/{key}", func(w http.ResponseWriter, r *http.Request) {
		s.handleRemove(w, r)

	})
	mu.HandleFunc("/clearall", func(w http.ResponseWriter, r *http.Request) {
		s.handleClearAll(w, r)
	})

	http.Serve(s.listener, mu)

}

func (s *Server) handleAll(w http.ResponseWriter, r *http.Request) {
	res := &Response{
		w:      w,
		Data:   nil,
		Status: 200,
	}
	res.Data = s.conns.GetAll()
	res.SendJson(1, "")
}

func (s *Server) handleClearAll(w http.ResponseWriter, r *http.Request) {
	res := &Response{
		w:      w,
		Data:   nil,
		Status: 200,
	}
	s.conns.RemoveAll()
	res.Data = struct{}{}
	res.SendJson(1, "")
}

func (s *Server) handleRemove(w http.ResponseWriter, r *http.Request) {
	res := &Response{
		w:      w,
		Data:   nil,
		Status: 200,
	}

	vars := mux.Vars(r)

	if key, ok := vars["key"]; ok {
		log.Println(key)
		s.conns.Remove(key)
		res.Data = struct{}{}
		res.SendJson(1, "")
	} else {
		res.Status = 404
		res.w.WriteHeader(res.Status)
	}
}

func (s *Server) handleSetList(w http.ResponseWriter, r *http.Request) {
	res := &Response{
		w:      w,
		Data:   nil,
		Status: 200,
	}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("[error]", "read request body :", err)
		return
	}
	list := []service.APIData{}
	err = json.Unmarshal(data, &list)
	if err != nil {
		log.Println("[error]", "json Unmarshal :", err, ". Request Data ", string(data))
		res.SendJson(0, "0002")
		return
	}

	go func() {
		for _, item := range list {
			s.conns.Add(item)
		}

	}()

	res.Data = struct{}{}
	res.SendJson(1, "")

}

func NewServer(ctx context.Context, port string) error {
	l, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}
	s := &Server{
		listener: l,
		conns:    service.NewConns(ctx),
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println("api server panic: ", r)
			}
		}()
		log.Println("api listen On ", port)
		s.Serve()
	}()
	return nil
}

type Server struct {
	listener net.Listener
	conns    *service.Conns
}
