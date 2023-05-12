package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/term"
)

type vplsInvoice struct {
	Id        string
	DateSent  string
	DateDue   string
	AmountDue string
}

type vplsCreditCard struct {
	Id   string
	Name string
}

func (card vplsCreditCard) String() string {
	return card.Id + "\t" + card.Name
}

func (invoice vplsInvoice) String() string {
	return invoice.Id + "\t" + invoice.DateSent + "\t" + invoice.DateDue + "\t" + invoice.AmountDue
}

const baseURL = "https://my.evocative.com"

var requiredCookies = []string{"evocative_session", "client_id", "evocative_token"}

func getInput(message string, hidden bool) string {
	var input string
	var err error
	count := 0
	log.Println(message)

	if hidden {
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatal(err)
		}

		input = string(bytePassword)
		count = len(input)
	} else {
		count, err = fmt.Scanln(&input)
		if err != nil {
			log.Fatal(err)
		}
	}

	if count < 1 {
		log.Println("Invalid input, try again")
		return getInput(message, hidden)
	}

	return input
}

func vplsChargeCard(client *http.Client, invoiceId, cardId string) {
	data := make(url.Values)
	data.Set("inv_id", invoiceId)
	data.Set("payment_method", "cc_"+cardId)

	resp, err := client.PostForm(baseURL+"/billing/charge_invoice", data)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	output, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusSeeOther {
		log.Fatal("Failed, expected status code " + strconv.Itoa(http.StatusSeeOther) + " but got status code: " + strconv.Itoa(resp.StatusCode) + "\n" + string(output))
	}

	time.Sleep(5 * time.Second)
	invoices := vplsListUnpaidInvoices(client)

	exists := false
	for _, x := range invoices {
		if x.Id == invoiceId {
			exists = true
			break
		}
	}

	if exists {
		log.Fatal("Invoice payment request sent but invoice still shows up in the list.\n " + string(output))
	}

	log.Println("Invoice successfully paid")
}

func vplsListCreditCards(client *http.Client) (cards []vplsCreditCard) {
	cards = []vplsCreditCard{}

	resp, err := client.Get(baseURL + "/settings/billing")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	output, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Fatal("status code: " + strconv.Itoa(resp.StatusCode) + "\n" + string(output))
	}

	currentEntry := vplsCreditCard{}
	tokenizer := html.NewTokenizer(strings.NewReader(string(output)))
	for {
		tt := tokenizer.Next()

		switch {

		case tt == html.ErrorToken:
			return

		case tt == html.StartTagToken:
			t := tokenizer.Token()
			if strings.HasPrefix(t.String(), "<button class=\"text-danger dropdown-item\" data-target=\"#del-credit-card_modal\" data-toggle=\"modal\"") {
				for _, x := range t.Attr {
					if x.Key == "data-cc_id" {
						currentEntry.Id = x.Val
					} else if x.Key == "data-cc_name" {
						currentEntry.Name = x.Val
						// this is the final record in the list, clear everything
						cards = append(cards, currentEntry)
						currentEntry = vplsCreditCard{}
					}
				}
			}
		}
	}
}

func vplsListUnpaidInvoices(client *http.Client) (invoices []vplsInvoice) {
	invoices = []vplsInvoice{}

	resp, err := client.Get(baseURL + "/billing/ajax_initial")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	output, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Fatal("status code: " + strconv.Itoa(resp.StatusCode) + "\n" + string(output))
	}

	var storeItem bool
	var itemType string

	currentEntry := vplsInvoice{}
	tokenizer := html.NewTokenizer(strings.NewReader(string(output)))
	for {
		tt := tokenizer.Next()

		switch {

		case tt == html.ErrorToken:
			return

		case tt == html.StartTagToken:
			t := tokenizer.Token()
			if strings.HasPrefix(t.String(), "<a href=\"https://my.evocative.com/billing/pay_invoice?inv_id=") {
				storeItem = true
				itemType = "invoice_id"
			} else if strings.HasPrefix(t.String(), "<td class=\"invoice-date-sent\">") {
				storeItem = true
				itemType = "invoice_date_sent"
			} else if strings.HasPrefix(t.String(), "<td class=\"invoice-date-due\">") {
				storeItem = true
				itemType = "invoice_date_due"
			} else if strings.HasPrefix(t.String(), "<td class=\"invoice-amount-due\">") {
				storeItem = true
				itemType = "invoice_amount_due"
			}

		case tt == html.TextToken:
			t := tokenizer.Token()
			if storeItem {
				if itemType == "invoice_id" {
					currentEntry.Id = strings.Split(t.String(), "-")[1]
				} else if itemType == "invoice_date_sent" {
					currentEntry.DateSent = t.String()
				} else if itemType == "invoice_date_due" {
					currentEntry.DateDue = t.String()
				} else if itemType == "invoice_amount_due" {
					currentEntry.AmountDue = t.String()
					// this is the final record in the list, clear everything
					invoices = append(invoices, currentEntry)
					currentEntry = vplsInvoice{}
				}
			}

			storeItem = false
			itemType = ""
		}
	}
}

func vplsLogin(client *http.Client, username, password string) {
	data := make(url.Values)
	data.Set("login", username)
	data.Set("password", password)

	resp, err := client.PostForm(baseURL+"/user/login", data)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	output, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Fatal("Failed, expected status code " + strconv.Itoa(http.StatusOK) + " but got status code: " + strconv.Itoa(resp.StatusCode) + "\n" + string(output))
	}

	cookies := client.Jar.Cookies(&url.URL{Scheme: "https", Host: "my.evocative.com"})

	if len(cookies) < len(requiredCookies) {
		log.Fatal("wrong number of cookies: " + strconv.Itoa(len(cookies)) + "\n" + string(output))
	}

	for _, x := range requiredCookies {
		found := false
		for _, cookie := range cookies {
			if x == cookie.Name {
				found = true
				break
			}
		}

		if !found {
			log.Fatal("could not find the cookie " + x)
		}
	}

}

func main() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(nil)
	}

	client := &http.Client{
		Jar: jar,
	}

	log.Println("Welcome to the program that allows you to pay your VPLS invoice! yes, seriously.")
	// login
	username := getInput("Username: ", false)
	password := getInput("Password: ", true)

	log.Println("Logging in...")
	vplsLogin(client, username, password)

	log.Println("Fetching unpaid invoices...")
	invoices := vplsListUnpaidInvoices(client)

	if len(invoices) == 0 {
		log.Fatal("No unpaid invoices found")
	}

	fmt.Println("# ID\tDate Sent\tDate Due\tAmount")
	for i, entry := range invoices {
		fmt.Println(strconv.Itoa(i+1), entry)
	}

	toPayInput := getInput("Invoice #: ", false)
	toPayInputInt, err := strconv.Atoi(toPayInput)
	if err != nil {
		log.Fatal(err)
	}
	if toPayInputInt > len(invoices) || toPayInputInt < 1 {
		log.Fatal("Invalid number selected")
	}

	selectedInvoice := invoices[toPayInputInt-1]

	log.Println("Fetching card list...")
	cards := vplsListCreditCards(client)

	if len(cards) == 0 {
		log.Fatal("No cards found")
	}

	fmt.Println("# ID\tName")
	for i, entry := range cards {
		fmt.Println(strconv.Itoa(i+1), entry)
	}

	cardInput := getInput("Card #: ", false)
	cardInputInt, err := strconv.Atoi(cardInput)
	if err != nil {
		log.Fatal(err)
	}
	if cardInputInt > len(cards) || cardInputInt < 1 {
		log.Fatal("Invalid number selected")
	}

	selectedCard := cards[cardInputInt-1]

	fmt.Println()
	fmt.Println()
	continueInput := strings.ToLower(getInput("The card '"+selectedCard.Name+"' will be charged for "+selectedInvoice.AmountDue+" (Invoice #"+selectedInvoice.Id+"), please confirm: y/N", false))
	if continueInput != "y" {
		log.Fatal("Aborted")
	}

	// submit request

	log.Println("Sending request...")
	vplsChargeCard(client, selectedInvoice.Id, selectedCard.Id)

}
