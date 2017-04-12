package main

import (
  "net/http"
  "net/url"
  "bytes"
  "errors"
  "encoding/json"
)

type LoginResponse struct {
  Ok bool
  Message string
  Session_id string
}

type HeartBeatResponse struct {
  Ok bool
  SystemNotification SystemNotification
  Self Self
  Message string
}

type SystemNotification struct {
  Show bool
  Link string
  Download_link string
  Version string
  Message string
}

type Self struct {
  Ssconfigs []Ssconfig
}

type Ssconfig struct {
  Id string
  Kcp bool
  Port int
  State int
  Method string
  Server string
  Remarks string
  Password string
  Fast_open bool
  Server_port string
}

func post(route string, fields url.Values) ([]byte, error) {
  resp, err := http.PostForm(API_HOST + route, fields)
  if err != nil {
    return nil, err
  }
  defer resp.Body.Close()
  buf := new(bytes.Buffer)
  buf.ReadFrom(resp.Body)
  return buf.Bytes(), nil
}

func RalletsLogin(username string, password string) (*LoginResponse, error) {
  fields := url.Values{}
  fields.Set("username_or_email", username)
  fields.Set("login_password", password)
  fields.Set("DEVICE_TYPE", DEVICE_TYPE)
  data, err := post("/login", fields)
  if err != nil {
    return nil, err
  }
  var resp LoginResponse
  json.Unmarshal(data, &resp)
  if !resp.Ok {
    return nil, errors.New(resp.Message)
  }
  return &resp, nil
}

func RalletsHeartBeat(sessionId string) (*HeartBeatResponse, error) {
  fields := url.Values{}
  fields.Set("session_id", sessionId)
  fields.Set("VERSION", VERSION)
  fields.Set("DEVICE_TYPE", DEVICE_TYPE)
  data, err := post("/rallets_notification", fields)
  if err != nil {
    return nil, err
  }
  var resp HeartBeatResponse
  json.Unmarshal(data, &resp)
  if !resp.Ok {
    return nil, errors.New(resp.Message)
  }
  return &resp, nil
}
