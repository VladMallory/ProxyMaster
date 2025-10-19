package common

type Client struct {
	ID         string `json:"id"`         // id клиента
	Email      string `json:"email"`      // email клиента
	Enable     bool   `json:"enable"`     // включен ли клиент
	ExpiryTime int64  `json:"expiryTime"` // время истечения доступа
	SubID      string `json:"subId"`      // id подписки
	Flow       string `json:"flow"`       // тип flow, например xtls
	LimitIP    int    `json:"limitIp"`    // лимит ip адресов
	TotalGB    int    `json:"totalGB"`    // лимит трафика
	Reset      int    `json:"reset"`      // период сброса данных
}

type Settings struct {
	Client     []Client `json:"clients"`    // слайс клиентов
	Decryption string   `json:"decryption"` // тип шифрования
}

// Запрос для входа
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Ответ от панели
type LoginResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

type Inbound struct {
	ID  int    `json:"success"`
	Msg string `json:""`
	Obj string `json:"obj"`
}

type InboundObj struct {
	ID             int    `json:"id"`
	Up             int64  `json:"up"`
	Down           int64  `json:"down"`
	Total          int64  `json:"total"`
	Remark         string `json:"remark"`
	Enable         bool   `json:"enable"`
	ExpiryTime     int64  `json:"expiryTime"`
	Listen         string `json:"listen"`
	Port           int    `json:"port"`
	Protocol       string `json:"protocol"`
	Settings       string `json:"settings"` // JSON-строка с настройками
	StreamSettings string `json:"streamSettings"`
	Tag            string `json:"tag"`
	Sniffing       string `json:"sniffing"`
}

type InboundInfo struct {
	Success bool       `json:"success"` // успешен ли запрос
	Msg     string     `json:"msg"`     // сообщение от сервера
	Obj     InboundObj `json:"obj"`     // json строка с данными
}
