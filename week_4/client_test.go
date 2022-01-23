package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	accessTokenOK = "qwerty"
	xmlDataPath   = "dataset.xml"
)

type Root struct {
	XMLName xml.Name `xml:"root"`
	Rows    []Row    `xml:"row"`
}

type Row struct {
	XMLName       xml.Name `xml:"row"`
	ID            string   `xml:"id"`
	Guid          string   `xml:"guid"`
	IsActive      string   `xml:"isActive"`
	Balance       string   `xml:"balance"`
	Picture       string   `xml:"picture"`
	Age           string   `xml:"age"`
	EyeColor      string   `xml:"eyeColor"`
	FirstName     string   `xml:"first_name"`
	LastName      string   `xml:"last_name"`
	Gender        string   `xml:"gender"`
	Company       string   `xml:"company"`
	Email         string   `xml:"email"`
	Phone         string   `xml:"phone"`
	Address       string   `xml:"address"`
	About         string   `xml:"about"`
	Registered    string   `xml:"registered"`
	FavoriteFruit string   `xml:"favoriteFruit"`
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("AccessToken") != accessTokenOK {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error": "wrong access token"}`)
		return
	}

	query := r.FormValue("query")
	switch query {
	case "__internal_error":
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"error": "internal server error"}`)
		return
	case "__wrong_error_json":
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `wrong error json`)
		return
	case "__unknown_bad_request":
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error": "unknown bad request"}`)
		return
	case "__timeout":
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusRequestTimeout)
		io.WriteString(w, `{"error": "timeout"}`)
		return
	}

	orderField := r.FormValue("order_field")
	if orderField != "Name" && orderField != "Age" && orderField != "Id" && orderField != "" {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error": "ErrorBadOrderField"}`)
		return
	}

	if query == "__cant_unpack_result_json" {
		io.WriteString(w, `cant unpack result json`)
		return
	}

	users, err := parseUsersFromXML(xmlDataPath)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error": "data could not be read"}`)
		return
	}

	if query != "" {
		users = filterUsers(users, query)
	}

	orderBy, err := parseOrderBy(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error": "the parameter 'order_by' could not be read"}`)
		return
	}

	if orderBy != OrderByAsIs {
		users = sortUsers(users, orderBy, orderField)
	}

	limit, err := parseLimit(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error": "the parameter 'limit' could not be read"}`)
		return
	}

	if limit == 0 || limit > len(users) {
		limit = len(users)
	}

	offset, err := parseOffset(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error": "the parameter 'offset' could not be read"}`)
		return
	}

	if limit+offset > len(users) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error": "wrong 'offset' parameter"}`)
		return
	}

	limitedResult := users[offset:(limit + offset)]

	bytes, err := json.Marshal(limitedResult)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error": "failed to marshal users"}`)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(bytes)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error": "failed to write result"}`)
		return
	}
}

func parseUsersFromXML(path string) ([]User, error) {
	xmlFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func(xmlFile *os.File) {
		err = xmlFile.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(xmlFile)

	byteValue, _ := ioutil.ReadAll(xmlFile)

	var root Root
	err = xml.Unmarshal(byteValue, &root)
	if err != nil {
		return nil, err
	}

	var users []User
	for _, row := range root.Rows {
		rowIDInt, err := strconv.Atoi(row.ID)
		if err != nil {
			return nil, err
		}

		rowAgeInt, err := strconv.Atoi(row.Age)
		if err != nil {
			return nil, err
		}

		user := User{
			Id:     rowIDInt,
			Name:   row.FirstName + " " + row.LastName,
			Age:    rowAgeInt,
			About:  row.About,
			Gender: row.Gender,
		}

		users = append(users, user)
	}

	return users, nil
}

func filterUsers(users []User, query string) []User {
	var c int
	for _, user := range users {
		if !strings.Contains(user.Name, query) && !strings.Contains(user.About, query) {
			continue
		}
		users[c] = user
		c++
	}
	return users[:c]
}

func sortUsers(users []User, orderBy int, orderField string) []User {
	switch orderField {
	case "Name", "":
		switch orderBy {
		case OrderByDesc:
			sort.Slice(users, func(i, j int) bool {
				return users[i].Name < users[j].Name
			})
			break
		case OrderByAsc:
			sort.Slice(users, func(i, j int) bool {
				return users[i].Name > users[j].Name
			})
			break
		}
		break
	case "Age":
		switch orderBy {
		case OrderByDesc:
			sort.Slice(users, func(i, j int) bool {
				return users[i].Age < users[j].Age
			})
			break
		case OrderByAsc:
			sort.Slice(users, func(i, j int) bool {
				return users[i].Age > users[j].Age
			})
			break
		}
		break
	case "Id":
		switch orderBy {
		case OrderByDesc:
			sort.Slice(users, func(i, j int) bool {
				return users[i].Id < users[j].Id
			})
			break
		case OrderByAsc:
			sort.Slice(users, func(i, j int) bool {
				return users[i].Id > users[j].Id
			})
			break
		}
		break
	}
	return users
}

func parseLimit(r *http.Request) (int, error) {
	limitStr := r.FormValue("limit")
	return strconv.Atoi(limitStr)
}

func parseOffset(r *http.Request) (int, error) {
	offsetStr := r.FormValue("offset")
	return strconv.Atoi(offsetStr)
}

func parseOrderBy(r *http.Request) (int, error) {
	orderBy := r.FormValue("order_by")
	return strconv.Atoi(orderBy)
}

func TestFindUsers_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         ts.URL,
		AccessToken: "",
	}

	searchRequest := SearchRequest{
		Limit:      0,
		Offset:     0,
		Query:      "",
		OrderField: "",
		OrderBy:    OrderByAsIs,
	}

	_, err := searchClient.FindUsers(searchRequest)
	if err == nil {
		t.Errorf("expected error: wrong access token")
	}
}

func TestFindUsers_LongTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         ts.URL,
		AccessToken: accessTokenOK,
	}

	searchRequest := SearchRequest{
		Limit:      0,
		Offset:     0,
		Query:      "__timeout",
		OrderField: "",
		OrderBy:    OrderByAsIs,
	}

	_, err := searchClient.FindUsers(searchRequest)
	if err == nil {
		t.Errorf("expected error: timeout")
	}
}

func TestFindUsers_WrongURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         "",
		AccessToken: accessTokenOK,
	}

	searchRequest := SearchRequest{
		Limit:      0,
		Offset:     0,
		Query:      "",
		OrderField: "",
		OrderBy:    OrderByAsIs,
	}

	_, err := searchClient.FindUsers(searchRequest)
	if err == nil {
		t.Errorf("expected error: uncorrect url")
	}
}

func TestFindUsers_InternalServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         ts.URL,
		AccessToken: accessTokenOK,
	}

	searchRequest := SearchRequest{
		Limit:      0,
		Offset:     0,
		Query:      "__internal_error",
		OrderField: "",
		OrderBy:    OrderByAsIs,
	}

	_, err := searchClient.FindUsers(searchRequest)
	if err == nil {
		t.Errorf("expected error: internal server error")
	}
}

func TestFindUsers_UnpackErrJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         ts.URL,
		AccessToken: accessTokenOK,
	}

	searchRequest := SearchRequest{
		Limit:      0,
		Offset:     0,
		Query:      "__wrong_error_json",
		OrderField: "",
		OrderBy:    OrderByAsIs,
	}

	_, err := searchClient.FindUsers(searchRequest)
	if err == nil {
		t.Errorf("expected error: cant unpack error json")
	}
}

func TestFindUsers_WrongOrderField(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         ts.URL,
		AccessToken: accessTokenOK,
	}

	searchRequest := SearchRequest{
		Limit:      0,
		Offset:     0,
		Query:      "",
		OrderField: "ERROR",
		OrderBy:    OrderByAsIs,
	}

	_, err := searchClient.FindUsers(searchRequest)
	if err == nil {
		t.Errorf("expected error: order field invalid")
	}
}

func TestFindUsers_UnknownBadRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         ts.URL,
		AccessToken: accessTokenOK,
	}

	searchRequest := SearchRequest{
		Limit:      0,
		Offset:     0,
		Query:      "__unknown_bad_request",
		OrderField: "",
		OrderBy:    OrderByAsIs,
	}

	_, err := searchClient.FindUsers(searchRequest)
	if err == nil {
		t.Errorf("expected error: unknown bad request error")
	}
}

func TestFindUsers_UnpackWrongJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         ts.URL,
		AccessToken: accessTokenOK,
	}

	searchRequest := SearchRequest{
		Limit:      0,
		Offset:     0,
		Query:      "__cant_unpack_result_json",
		OrderField: "",
		OrderBy:    OrderByAsIs,
	}

	_, err := searchClient.FindUsers(searchRequest)
	if err == nil {
		t.Errorf("expected error: cant unpack result json")
	}
}

func TestFindUsers_SortAgeByDesc(t *testing.T) {
	expectedUsers := []User{{Name: "Terrell Hall"}, {Name: "Gilmore Guerra"}, {Name: "Cruz Guerrero"}}

	expectedUserCount := len(expectedUsers)

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         ts.URL,
		AccessToken: accessTokenOK,
	}

	searchRequest := SearchRequest{
		Limit:      3,
		Offset:     0,
		Query:      "err",
		OrderField: "Age",
		OrderBy:    OrderByDesc,
	}

	response, err := searchClient.FindUsers(searchRequest)
	if err != nil {
		t.Errorf("unexpected err while finding users")
		return
	}

	if response == nil {
		t.Errorf("response must not be nil")
		return
	}

	if len(response.Users) != expectedUserCount {
		t.Errorf("user count from response not equals to expected user count")
		return
	}

	for idx, user := range response.Users {
		if user.Name != expectedUsers[idx].Name {
			t.Errorf("wrong result")
			return
		}
	}
}

func TestFindUsers_SortAgeByAsc(t *testing.T) {
	expectedUsers := []User{{Name: "Jennings Mays"}, {Name: "Annie Osborn"}, {Name: "Leann Travis"}}

	expectedUserCount := len(expectedUsers)

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         ts.URL,
		AccessToken: accessTokenOK,
	}

	searchRequest := SearchRequest{
		Limit:      3,
		Offset:     0,
		Query:      "nn",
		OrderField: "Age",
		OrderBy:    OrderByAsc,
	}

	response, err := searchClient.FindUsers(searchRequest)
	if err != nil {
		t.Errorf("unexpected err while finding users")
		return
	}

	if response == nil {
		t.Errorf("response must not be nil")
		return
	}

	if len(response.Users) != expectedUserCount {
		t.Errorf("user count from response not equals to expected user count")
		return
	}

	for idx, user := range response.Users {
		if user.Name != expectedUsers[idx].Name {
			t.Errorf("wrong result")
			return
		}
	}
}

func TestFindUsers_LargeLimit(t *testing.T) {
	expectedUserCount := 25

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         ts.URL,
		AccessToken: accessTokenOK,
	}

	searchRequest := SearchRequest{
		Limit:      30,
		Offset:     0,
		Query:      "",
		OrderField: "",
		OrderBy:    OrderByAsIs,
	}

	response, err := searchClient.FindUsers(searchRequest)
	if err != nil {
		t.Errorf("unexpected err while finding users")
		return
	}

	if response == nil {
		t.Errorf("response must not be nil")
		return
	}

	if len(response.Users) != expectedUserCount {
		t.Errorf("user count from response not equals to expected user count")
		return
	}
}

func TestFindUsers_WithoutQuery(t *testing.T) {
	expectedUserCount := 15

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         ts.URL,
		AccessToken: accessTokenOK,
	}

	searchRequest := SearchRequest{
		Limit:      15,
		Offset:     0,
		Query:      "",
		OrderField: "",
		OrderBy:    OrderByAsIs,
	}

	response, err := searchClient.FindUsers(searchRequest)
	if err != nil {
		t.Errorf("unexpected err while finding users")
		return
	}

	if response == nil {
		t.Errorf("response must not be nil")
		return
	}

	if len(response.Users) != expectedUserCount {
		t.Errorf("user count from response not equals to expected user count")
		return
	}
}

func TestFindUsers_NegativeLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         ts.URL,
		AccessToken: accessTokenOK,
	}

	searchRequest := SearchRequest{
		Limit:      -1,
		Offset:     0,
		Query:      "",
		OrderField: "",
		OrderBy:    OrderByAsIs,
	}

	_, err := searchClient.FindUsers(searchRequest)
	if err == nil {
		t.Errorf("expected error with limit")
		return
	}
}

func TestFindUsers_NegativeOffset(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	searchClient := SearchClient{
		URL:         ts.URL,
		AccessToken: accessTokenOK,
	}

	searchRequest := SearchRequest{
		Limit:      0,
		Offset:     -1,
		Query:      "",
		OrderField: "",
		OrderBy:    OrderByAsIs,
	}

	_, err := searchClient.FindUsers(searchRequest)
	if err == nil {
		t.Errorf("expected error with offset")
		return
	}
}
