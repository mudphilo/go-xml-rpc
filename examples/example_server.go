package main

import (
    "encoding/json"
    "github.com/mudphilo/go-xml-rpc"
    "github.com/mudphilo/go-xml-rpc/xml"
    "log"
    "net/http"
)

type UssdRequest struct {
    UserId       string `xml:"USER_ID" json:"user_id"`
    UserPassword string `xml:"USER_PASSWORD" json:"user_password"`
    Sequence     string `xml:"SEQUENCE" json:"sequence"`
    EndOfSession string `xml:"END_OF_SESSION" json:"end_of_session"`
    Language    string `xml:"LANGUAGE" json:"language"`
    SessionId     string `xml:"SESSION_ID" json:"session_id"`
    ServiceKey   string `xml:"SERVICE_KEY" json:"service_key"`
    MobileNumber string `xml:"MOBILE_NUMBER" json:"mobile_number"`
    Imsi     string `xml:"IMSI" json:"imsi"`
    UssdBody string `xml:"USSD_BODY" json:"ussd_body"`
}

type UssdResponse struct {

    RESPONSE_CODE string `xml:"RESPONSE_CODE"`
    REQUEST_TYPE string `xml:"REQUEST_TYPE"`
    SESSION_ID string `xml:"SESSION_ID"`
    SEQUENCE string `xml:"SEQUENCE"`
    USSD_BODY string `xml:"USSD_BODY"`
    END_OF_SESSION string `xml:"END_OF_SESSION"`
}

type HelloService struct{}

func (h *HelloService) UssdMessage(r *http.Request, args *UssdRequest, reply *UssdResponse) error {

    jD, _ := json.Marshal(args)
    log.Printf("USSD_MESSAGE %s",string(jD))

    reply.END_OF_SESSION = "session here"
    reply.REQUEST_TYPE = "USSD"
    reply.SEQUENCE = args.Sequence
    reply.RESPONSE_CODE = args.SessionId
    reply.USSD_BODY = args.UssdBody
    return nil
}

func (h *HelloService) Say(r *http.Request, args *struct{Who string}, reply *struct{Message string}) error {

    log.Println("Say", args.Who)
    reply.Message = "Hello, " + args.Who + "!"
    return nil
}

func main() {

    RPC := rpc.NewServer()
    xmlrpcCodec := xml.NewCodec()
    RPC.RegisterCodec(xmlrpcCodec, "text/xml")
    err := RPC.RegisterDefaultService(new(HelloService), "")
    if err != nil {

        panic(err)
    }

    http.Handle("/ussd", RPC)
    log.Println("Starting XML-RPC server on localhost:1234/ussd")
    log.Fatal(http.ListenAndServe(":1234", nil))
}