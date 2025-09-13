package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestLogin_Success тестирует успешную авторизацию в панели
func TestLogin_Success(t *testing.T) {
	// Создаем mock сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что запрос идет на правильный endpoint
		if r.URL.Path != "/login" {
			t.Errorf("Ожидался путь /login, получен %s", r.URL.Path)
		}

		// Проверяем метод запроса
		if r.Method != "POST" {
			t.Errorf("Ожидался метод POST, получен %s", r.Method)
		}

		// Проверяем Content-Type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Ожидался Content-Type application/json, получен %s", contentType)
		}

		// Читаем тело запроса
		var loginReq LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
			t.Errorf("Ошибка декодирования запроса: %v", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Проверяем данные авторизации
		expectedUsername := "test_user"
		expectedPassword := "test_password"
		if loginReq.Username != expectedUsername {
			t.Errorf("Ожидался username %s, получен %s", expectedUsername, loginReq.Username)
		}
		if loginReq.Password != expectedPassword {
			t.Errorf("Ожидался password %s, получен %s", expectedPassword, loginReq.Password)
		}

		// Отправляем успешный ответ
		response := LoginResponse{
			Success: true,
			Msg:     "Login successful",
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Set-Cookie", "3x-ui=test_session_cookie; Path=/; HttpOnly")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Сохраняем оригинальные значения конфигурации
	originalPanelURL := PANEL_URL
	originalPanelUser := PANEL_USER
	originalPanelPass := PANEL_PASS

	// Устанавливаем тестовые значения
	PANEL_URL = server.URL + "/"
	PANEL_USER = "test_user"
	PANEL_PASS = "test_password"

	// Восстанавливаем оригинальные значения после теста
	defer func() {
		PANEL_URL = originalPanelURL
		PANEL_USER = originalPanelUser
		PANEL_PASS = originalPanelPass
	}()

	// Выполняем тест
	sessionCookie, err := Login()

	// Проверяем результат
	if err != nil {
		t.Errorf("Login() вернул ошибку: %v", err)
	}

	expectedCookie := "3x-ui=test_session_cookie"
	if sessionCookie != expectedCookie {
		t.Errorf("Ожидалась кука %s, получена %s", expectedCookie, sessionCookie)
	}
}

// TestLogin_InvalidCredentials тестирует авторизацию с неверными данными
func TestLogin_InvalidCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := LoginResponse{
			Success: false,
			Msg:     "Invalid credentials",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Сохраняем оригинальные значения
	originalPanelURL := PANEL_URL
	originalPanelUser := PANEL_USER
	originalPanelPass := PANEL_PASS

	// Устанавливаем тестовые значения
	PANEL_URL = server.URL + "/"
	PANEL_USER = "wrong_user"
	PANEL_PASS = "wrong_password"

	// Восстанавливаем оригинальные значения после теста
	defer func() {
		PANEL_URL = originalPanelURL
		PANEL_USER = originalPanelUser
		PANEL_PASS = originalPanelPass
	}()

	// Выполняем тест
	sessionCookie, err := Login()

	// Проверяем, что вернулась ошибка
	if err == nil {
		t.Error("Login() должен был вернуть ошибку для неверных данных")
	}

	if sessionCookie != "" {
		t.Errorf("SessionCookie должен быть пустым, получен %s", sessionCookie)
	}

	// Проверяем текст ошибки
	expectedErrorMsg := "авторизация не удалась: Invalid credentials"
	if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Ожидалась ошибка содержащая '%s', получена: %s", expectedErrorMsg, err.Error())
	}
}

// TestLogin_ServerError тестирует случай ошибки сервера
func TestLogin_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	// Сохраняем оригинальные значения
	originalPanelURL := PANEL_URL
	originalPanelUser := PANEL_USER
	originalPanelPass := PANEL_PASS

	// Устанавливаем тестовые значения
	PANEL_URL = server.URL + "/"
	PANEL_USER = "test_user"
	PANEL_PASS = "test_password"

	// Восстанавливаем оригинальные значения после теста
	defer func() {
		PANEL_URL = originalPanelURL
		PANEL_USER = originalPanelUser
		PANEL_PASS = originalPanelPass
	}()

	// Выполняем тест
	sessionCookie, err := Login()

	// Проверяем, что вернулась ошибка
	if err == nil {
		t.Error("Login() должен был вернуть ошибку при ошибке сервера")
	}

	if sessionCookie != "" {
		t.Errorf("SessionCookie должен быть пустым, получен %s", sessionCookie)
	}

	// Проверяем текст ошибки
	expectedErrorMsg := "некорректный статус ответа: 500"
	if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Ожидалась ошибка содержащая '%s', получена: %s", expectedErrorMsg, err.Error())
	}
}

// TestLogin_EmptyResponse тестирует случай пустого ответа от сервера
func TestLogin_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Отправляем пустой ответ
	}))
	defer server.Close()

	// Сохраняем оригинальные значения
	originalPanelURL := PANEL_URL
	originalPanelUser := PANEL_USER
	originalPanelPass := PANEL_PASS

	// Устанавливаем тестовые значения
	PANEL_URL = server.URL + "/"
	PANEL_USER = "test_user"
	PANEL_PASS = "test_password"

	// Восстанавливаем оригинальные значения после теста
	defer func() {
		PANEL_URL = originalPanelURL
		PANEL_USER = originalPanelUser
		PANEL_PASS = originalPanelPass
	}()

	// Выполняем тест
	sessionCookie, err := Login()

	// Проверяем, что вернулась ошибка
	if err == nil {
		t.Error("Login() должен был вернуть ошибку при пустом ответе")
	}

	if sessionCookie != "" {
		t.Errorf("SessionCookie должен быть пустым, получен %s", sessionCookie)
	}

	// Проверяем текст ошибки
	expectedErrorMsg := "пустой ответ от сервера"
	if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Ожидалась ошибка содержащая '%s', получена: %s", expectedErrorMsg, err.Error())
	}
}

// TestLogin_InvalidJSON тестирует случай невалидного JSON ответа
func TestLogin_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json response"))
	}))
	defer server.Close()

	// Сохраняем оригинальные значения
	originalPanelURL := PANEL_URL
	originalPanelUser := PANEL_USER
	originalPanelPass := PANEL_PASS

	// Устанавливаем тестовые значения
	PANEL_URL = server.URL + "/"
	PANEL_USER = "test_user"
	PANEL_PASS = "test_password"

	// Восстанавливаем оригинальные значения после теста
	defer func() {
		PANEL_URL = originalPanelURL
		PANEL_USER = originalPanelUser
		PANEL_PASS = originalPanelPass
	}()

	// Выполняем тест
	sessionCookie, err := Login()

	// Проверяем, что вернулась ошибка
	if err == nil {
		t.Error("Login() должен был вернуть ошибку при невалидном JSON")
	}

	if sessionCookie != "" {
		t.Errorf("SessionCookie должен быть пустым, получен %s", sessionCookie)
	}

	// Проверяем текст ошибки
	expectedErrorMsg := "ошибка десериализации ответа"
	if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Ожидалась ошибка содержащая '%s', получена: %s", expectedErrorMsg, err.Error())
	}
}

// TestLogin_NoSessionCookie тестирует случай, когда сервер не возвращает куку сессии
func TestLogin_NoSessionCookie(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := LoginResponse{
			Success: true,
			Msg:     "Login successful",
		}

		w.Header().Set("Content-Type", "application/json")
		// Не устанавливаем Set-Cookie заголовок
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Сохраняем оригинальные значения
	originalPanelURL := PANEL_URL
	originalPanelUser := PANEL_USER
	originalPanelPass := PANEL_PASS

	// Устанавливаем тестовые значения
	PANEL_URL = server.URL + "/"
	PANEL_USER = "test_user"
	PANEL_PASS = "test_password"

	// Восстанавливаем оригинальные значения после теста
	defer func() {
		PANEL_URL = originalPanelURL
		PANEL_USER = originalPanelUser
		PANEL_PASS = originalPanelPass
	}()

	// Выполняем тест
	sessionCookie, err := Login()

	// Проверяем, что вернулась ошибка
	if err == nil {
		t.Error("Login() должен был вернуть ошибку при отсутствии куки сессии")
	}

	if sessionCookie != "" {
		t.Errorf("SessionCookie должен быть пустым, получен %s", sessionCookie)
	}

	// Проверяем текст ошибки
	expectedErrorMsg := "кука сессии не найдена"
	if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Ожидалась ошибка содержащая '%s', получена: %s", expectedErrorMsg, err.Error())
	}
}

// TestLogin_NetworkError тестирует случай сетевой ошибки
func TestLogin_NetworkError(t *testing.T) {
	// Создаем сервер, который сразу закрывается
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ничего не делаем
	}))
	server.Close() // Закрываем сервер сразу

	// Сохраняем оригинальные значения
	originalPanelURL := PANEL_URL
	originalPanelUser := PANEL_USER
	originalPanelPass := PANEL_PASS

	// Устанавливаем тестовые значения
	PANEL_URL = server.URL + "/"
	PANEL_USER = "test_user"
	PANEL_PASS = "test_password"

	// Восстанавливаем оригинальные значения после теста
	defer func() {
		PANEL_URL = originalPanelURL
		PANEL_USER = originalPanelUser
		PANEL_PASS = originalPanelPass
	}()

	// Выполняем тест
	sessionCookie, err := Login()

	// Проверяем, что вернулась ошибка
	if err == nil {
		t.Error("Login() должен был вернуть ошибку при сетевой ошибке")
	}

	if sessionCookie != "" {
		t.Errorf("SessionCookie должен быть пустым, получен %s", sessionCookie)
	}

	// Проверяем текст ошибки
	expectedErrorMsg := "ошибка выполнения запроса"
	if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Ожидалась ошибка содержащая '%s', получена: %s", expectedErrorMsg, err.Error())
	}
}

// TestLogin_Timeout тестирует случай таймаута
func TestLogin_Timeout(t *testing.T) {
	// Пропускаем этот тест в обычном режиме, так как он занимает много времени
	t.Skip("Пропуск теста таймаута (занимает 35 секунд)")

	// Создаем сервер, который отвечает с задержкой больше таймаута
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ждем больше 30 секунд (таймаут в httpClient)
		time.Sleep(35 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Сохраняем оригинальные значения
	originalPanelURL := PANEL_URL
	originalPanelUser := PANEL_USER
	originalPanelPass := PANEL_PASS

	// Устанавливаем тестовые значения
	PANEL_URL = server.URL + "/"
	PANEL_USER = "test_user"
	PANEL_PASS = "test_password"

	// Восстанавливаем оригинальные значения после теста
	defer func() {
		PANEL_URL = originalPanelURL
		PANEL_USER = originalPanelUser
		PANEL_PASS = originalPanelPass
	}()

	// Выполняем тест
	sessionCookie, err := Login()

	// Проверяем, что вернулась ошибка
	if err == nil {
		t.Error("Login() должен был вернуть ошибку при таймауте")
	}

	if sessionCookie != "" {
		t.Errorf("SessionCookie должен быть пустым, получен %s", sessionCookie)
	}

	// Проверяем текст ошибки
	expectedErrorMsg := "ошибка выполнения запроса"
	if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Ожидалась ошибка содержащая '%s', получена: %s", expectedErrorMsg, err.Error())
	}
}

// TestLogin_Integration тестирует интеграцию с реальными настройками из config.go
func TestLogin_Integration(t *testing.T) {
	// Пропускаем тест, если не установлены переменные окружения для интеграционного тестирования
	if testing.Short() {
		t.Skip("Пропуск интеграционного теста в коротком режиме")
	}

	// Проверяем, что у нас есть реальные настройки из config.go
	if PANEL_URL == "" || PANEL_USER == "" || PANEL_PASS == "" {
		t.Skip("Пропуск интеграционного теста: не настроены переменные PANEL_URL, PANEL_USER или PANEL_PASS")
	}

	// Логируем используемые настройки для отладки
	t.Logf("Интеграционный тест: URL=%s, User=%s", PANEL_URL, PANEL_USER)

	// Выполняем реальный запрос к настоящей панели
	sessionCookie, err := Login()

	// В интеграционном тесте мы можем получить как успех, так и ошибку
	// Главное - проверить, что функция работает корректно
	if err != nil {
		// Если ошибка, проверяем, что это разумная ошибка
		errorMsg := err.Error()
		validErrors := []string{
			"ошибка выполнения запроса",
			"некорректный статус ответа",
			"авторизация не удалась",
			"ошибка десериализации ответа",
			"пустой ответ от сервера",
			"кука сессии не найдена",
		}

		hasValidError := false
		for _, validError := range validErrors {
			if strings.Contains(errorMsg, validError) {
				hasValidError = true
				break
			}
		}

		if !hasValidError {
			t.Errorf("Получена неожиданная ошибка: %s", errorMsg)
		}

		// В случае ошибки sessionCookie должен быть пустым
		if sessionCookie != "" {
			t.Errorf("SessionCookie должен быть пустым при ошибке, получен: %s", sessionCookie)
		}
	} else {
		// Если успех, проверяем, что sessionCookie не пустой и содержит правильный формат
		if sessionCookie == "" {
			t.Error("SessionCookie не должен быть пустым при успешной авторизации")
		}

		// Проверяем формат куки (должна содержать "3x-ui=")
		if !strings.Contains(sessionCookie, "3x-ui=") {
			t.Errorf("SessionCookie должен содержать '3x-ui=', получен: %s", sessionCookie)
		}

		t.Logf("Интеграционный тест успешен: получена кука %s", sessionCookie)
	}
}

// TestConfigVariables тестирует, что переменные конфигурации корректно загружены
func TestConfigVariables(t *testing.T) {
	// Проверяем, что основные переменные не пустые
	if PANEL_URL == "" {
		t.Error("PANEL_URL не должен быть пустым")
	}
	if PANEL_USER == "" {
		t.Error("PANEL_USER не должен быть пустым")
	}
	if PANEL_PASS == "" {
		t.Error("PANEL_PASS не должен быть пустым")
	}
	if INBOUND_ID <= 0 {
		t.Error("INBOUND_ID должен быть больше 0")
	}

	// Логируем значения для отладки (без пароля)
	t.Logf("Конфигурация загружена: URL=%s, User=%s, InboundID=%d",
		PANEL_URL, PANEL_USER, INBOUND_ID)

	// Проверяем, что URL выглядит как валидный
	if !strings.HasPrefix(PANEL_URL, "http://") && !strings.HasPrefix(PANEL_URL, "https://") {
		t.Errorf("PANEL_URL должен начинаться с http:// или https://, получен: %s", PANEL_URL)
	}
}
