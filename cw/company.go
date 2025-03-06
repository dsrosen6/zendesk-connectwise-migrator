package cw

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

type Companies []Company
type Company struct {
	Id         int    `json:"id"`
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
	Status     struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			StatusHref string `json:"status_href"`
		} `json:"_info"`
	} `json:"status"`
	AddressLine1 string `json:"addressLine1"`
	AddressLine2 string `json:"addressLine2"`
	City         string `json:"city"`
	Zip          string `json:"zip"`
	Country      struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			CountryHref string `json:"country_href"`
		} `json:"_info"`
	} `json:"country"`
	PhoneNumber string `json:"phoneNumber"`
	FaxNumber   string `json:"faxNumber"`
	Website     string `json:"website"`
	Territory   struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			LocationHref string `json:"location_href"`
		} `json:"_info"`
	} `json:"territory"`
	DefaultContact struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			ContactHref string `json:"contact_href"`
		} `json:"_info"`
	} `json:"defaultContact"`
	DateAcquired    time.Time `json:"dateAcquired"`
	AnnualRevenue   float64   `json:"annualRevenue"`
	LeadFlag        bool      `json:"leadFlag"`
	UnsubscribeFlag bool      `json:"unsubscribeFlag"`
	TaxCode         struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			TaxCodeHref string `json:"taxCode_href"`
		} `json:"_info"`
	} `json:"taxCode"`
	BillToCompany struct {
		Id         int    `json:"id"`
		Identifier string `json:"identifier"`
		Name       string `json:"name"`
		Info       struct {
			CompanyHref string `json:"company_href"`
		} `json:"_info"`
	} `json:"billToCompany"`
	BillingSite struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			SiteHref string `json:"site_href"`
		} `json:"_info"`
	} `json:"billingSite"`
	BillingContact struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			ContactHref string `json:"contact_href"`
		} `json:"_info"`
	} `json:"billingContact"`
	InvoiceToEmailAddress string `json:"invoiceToEmailAddress"`
	DeletedFlag           bool   `json:"deletedFlag"`
	MobileGuid            string `json:"mobileGuid"`
	TerritoryManager      struct {
		Id         int    `json:"id"`
		Identifier string `json:"identifier"`
		Name       string `json:"name"`
		Info       struct {
			MemberHref string `json:"member_href"`
		} `json:"_info"`
	} `json:"territoryManager"`
	IsVendorFlag bool `json:"isVendorFlag"`
	Types        []struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			TypeHref string `json:"type_href"`
		} `json:"_info"`
	} `json:"types"`
	Site struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
		Info struct {
			SiteHref string `json:"site_href"`
		} `json:"_info"`
	} `json:"site"`
	Info struct {
		LastUpdated        time.Time `json:"lastUpdated"`
		UpdatedBy          string    `json:"updatedBy"`
		DateEntered        time.Time `json:"dateEntered"`
		EnteredBy          string    `json:"enteredBy"`
		ContactsHref       string    `json:"contacts_href"`
		AgreementsHref     string    `json:"agreements_href"`
		TicketsHref        string    `json:"tickets_href"`
		OpportunitiesHref  string    `json:"opportunities_href"`
		ActivitiesHref     string    `json:"activities_href"`
		ProjectsHref       string    `json:"projects_href"`
		ConfigurationsHref string    `json:"configurations_href"`
		OrdersHref         string    `json:"orders_href"`
		DocumentsHref      string    `json:"documents_href"`
		SitesHref          string    `json:"sites_href"`
		TeamsHref          string    `json:"teams_href"`
		ReportsHref        string    `json:"reports_href"`
		NotesHref          string    `json:"notes_href"`
	} `json:"_info"`
	CustomFields []struct {
		Id               int    `json:"id"`
		Caption          string `json:"caption"`
		Type             string `json:"type"`
		EntryMethod      string `json:"entryMethod"`
		NumberOfDecimals int    `json:"numberOfDecimals"`
		Value            string `json:"value"`
		ConnectWiseId    string `json:"connectWiseId"`
	} `json:"customFields"`
}

func (c *Client) GetCompanyByName(ctx context.Context, name string) (Company, error) {
	query := url.QueryEscape(fmt.Sprintf("name=\"%s\"", name))
	u := fmt.Sprintf("%s/company/companies?conditions=%s", baseUrl, query)
	co := Companies{}

	if err := c.apiRequest(ctx, "GET", u, nil, &co); err != nil {
		return Company{}, fmt.Errorf("an error occured getting the company: %w", err)
	}

	if len(co) != 1 {
		return Company{}, fmt.Errorf("expected 1 company, got %d", len(co))
	}

	return co[0], nil
}
