package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const API_CEP_BASE_URL = "https://cdn.apicep.com/file/apicep"
const VIACEP_BASE_URL = "http://viacep.com.br/ws"

type ApiCepData struct {
	Status   int    `json:"status"`
	Code     string `json:"code"`
	State    string `json:"state"`
	City     string `json:"city"`
	District string `json:"district"`
	Address  string `json:"address"`
}

type ViaCepData struct {
	Cep         string `json:"cep"`
	Logradouro  string `json:"logradouro"`
	Complemento string `json:"complemento"`
	Bairro      string `json:"bairro"`
	Localidade  string `json:"localidade"`
	Uf          string `json:"uf"`
	Ibge        string `json:"ibge"`
	Gia         string `json:"gia"`
	Ddd         string `json:"ddd"`
	Siafi       string `json:"siafi"`
}

type CepData interface {
	ApiCepData | ViaCepData
}

func makeRequest[T CepData](ctx context.Context, cep, baseUrl string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", baseUrl, nil)

	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 && resp.StatusCode <= 599 {
		return nil, errors.New(string(body))
	}

	var cepData T
	err = json.Unmarshal(body, &cepData)

	if err != nil {
		return nil, err
	}

	return &cepData, nil
}

func getApiCep(ctx context.Context, cep string, cepCh chan<- ApiCepData) {
	data, err := makeRequest[ApiCepData](ctx, cep, fmt.Sprintf("%s/%s.json", API_CEP_BASE_URL, cep))

	if err != nil {
		log.Printf("unable to get the cep data from 'apicep': %s", err)
		return
	}

	cepCh <- *data
}

func getViaCep(ctx context.Context, cep string, cepCh chan ViaCepData) {
	data, err := makeRequest[ViaCepData](ctx, cep, fmt.Sprintf("%s/%s/json", VIACEP_BASE_URL, cep))

	if err != nil {
		log.Printf("unable to get the cep data from 'viacep': %s", err)
		return
	}

	cepCh <- *data
}

func handleResponseReceived[T CepData](api string, data T) {
	dataJson, err := json.Marshal(data)

	if err != nil {
		log.Fatalf("unable to serialize the cep data into json: %s", err)
	}

	fmt.Printf("response received from the '%s' api. Response data: %s\n", api, string(dataJson))
}

func main() {
	timeoutDuration := time.Second * 1

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, timeoutDuration)

	defer cancel()

	apiCepCh := make(chan ApiCepData)
	viaCepCh := make(chan ViaCepData)

	go getApiCep(ctx, "06233-030", apiCepCh)
	go getViaCep(ctx, "06233030", viaCepCh)

	select {
	case cepData := <-apiCepCh:
		handleResponseReceived[ApiCepData]("apicep", cepData)
	case cepData := <-viaCepCh:
		handleResponseReceived[ViaCepData]("viacep", cepData)
	case <-time.After(timeoutDuration):
		log.Println("the request didn't finish, timeout exceeded")
	}
}
