package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"limit_traffic/common"
	"log"
	"net/http"
	"strings"
	"time"
)

// htpp клиент
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func main() {
	start := time.Now()
	log.Println("Вход в панель")

	// данные для входа
	loginData := common.LoginRequest{
		Username: common.Panel_User,
		Password: common.Panel_Pass,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		fmt.Println(err)
	}

	// запрос который будем отправлять для подключения
	req, err := http.NewRequest("POST", common.PANEL_URL+"login", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal("Ошибка отправки:", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// отправляем запрос на подключение
	log.Println("Отправка запроса...")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatal("Ошибка отправки запроса:", err)
	}

	// обработка ошибки соединения
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Fatal("Ошибка закртия соединения:", err)
		}
	}()

	// читаем ответ
	body, err := io.ReadAll(resp.Body)

	// парсим json
	var loginResp common.LoginResponse
	var cookiesResult []string

	if err := json.Unmarshal(body, &loginResp); err != nil {
		log.Fatal("Вход не удался", loginResp.Msg)
	}

	// ищем куки
	log.Println("\nПоиск coockie в заголовках...")
	var sessionCookie string
	found := false

	for _, cookie := range resp.Header.Values("Set-Cookie") {
		if strings.Contains(cookie, "3x-ui=") {
			sessionCookie = strings.Split(cookie, ";")[0]
			cookiesResult = append(cookiesResult, sessionCookie)
			found = true
		}
	}

	// вывод cookie в консоль
	for _, c := range cookiesResult {
		fmt.Println("Успешный вход. Cookie: ", c[:20], "...")
	}

	// если сервер не дал cookie
	if !found {
		log.Fatal("Cookie не получена", found)
	}

	// ===ПОЛУЧАЕМ КЛИЕНТОВ===
	log.Println("Получения списка клиентов")

	// GET запрос для получения инбаунда
	url := fmt.Sprintf("%spanel/api/inbounds/get/%d", common.PANEL_URL, common.Inbound_Id)
	req2, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("Ошибка создания запроса: ", err)
	}

	// добавление cookie для авторизации
	req2.Header.Set("Cookie", sessionCookie)

	// отправляем запрос для получения inbaund
	resp2, err := httpClient.Do(req2)
	if err != nil {
		log.Fatal("Ошибка в отправке запроса", err)
	}

	// обработка закрытия запроса
	defer func() {
		if err := resp2.Body.Close(); err != nil {
			log.Fatal("Ошибка закрытия запроса", err)
		}
	}()

	// чтение ответа
	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		log.Fatal("Ошибка чтения ответа:", err)
	}

	// парсинг json ответа
	var inbaundInfo common.InboundInfo
	if err := json.Unmarshal(body2, &inbaundInfo); err != nil {
		log.Fatal("Ошибка парсинга json: ", err)
	}

	if !inbaundInfo.Success {
		log.Fatal("Ошибка от сервера: ", inbaundInfo.Msg)
	}

	var settings common.Settings
	if err := json.Unmarshal([]byte(inbaundInfo.Obj.Settings), &settings); err != nil {
		log.Fatal("Ошибка парсинга настроек: ", err)
	}

	// список клиентов
	log.Printf("Получено клиентов: %d\n", len(settings.Client))

	// вывод информации о клиентах
	for _, client := range settings.Client {
		fmt.Printf("Клиент: %s, Email: %s, Включен: %v, Трафик: %d GB\n",
			client.ID, client.Email, client.Enable, client.TotalGB)
	}

	// замер скорости программы
	end := time.Since(start)
	fmt.Println("Время выполнения:", end)
}
