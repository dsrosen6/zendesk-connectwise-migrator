package psa

import "time"

type Company struct {
	Id                    int            `json:"id,omitempty"`
	Identifier            string         `json:"identifier,omitempty"`
	Name                  string         `json:"name,omitempty"`
	AddressLine1          string         `json:"addressLine1,omitempty"`
	AddressLine2          string         `json:"addressLine2,omitempty"`
	City                  string         `json:"city,omitempty"`
	Zip                   string         `json:"zip,omitempty"`
	Country               Country        `json:"country,omitempty"`
	PhoneNumber           string         `json:"phoneNumber,omitempty"`
	FaxNumber             string         `json:"faxNumber,omitempty"`
	Website               string         `json:"website,omitempty"`
	Territory             Territory      `json:"territory,omitempty"`
	DefaultContact        DefaultContact `json:"defaultContact,omitempty"`
	DateAcquired          time.Time      `json:"dateAcquired,omitempty"`
	AnnualRevenue         float64        `json:"annualRevenue,omitempty"`
	LeadFlag              bool           `json:"leadFlag,omitempty"`
	UnsubscribeFlag       bool           `json:"unsubscribeFlag,omitempty"`
	InvoiceToEmailAddress string         `json:"invoiceToEmailAddress,omitempty"`
	DeletedFlag           bool           `json:"deletedFlag,omitempty"`
	MobileGuid            string         `json:"mobileGuid,omitempty"`
	IsVendorFlag          bool           `json:"isVendorFlag,omitempty"`
	Types                 []Type         `json:"types,omitempty"`
	Site                  Site           `json:"site,omitempty"`
}

type ContactPostBody struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Company   struct {
		Id int `json:"id"`
	} `json:"company"`
	CommunicationItems []CommunicationItem `json:"communicationItems"`
}

type Contact struct {
	Id                 int                 `json:"id,omitempty"`
	FirstName          string              `json:"firstName,omitempty"`
	LastName           string              `json:"lastName,omitempty"`
	Company            Company             `json:"company,omitempty"`
	Site               Site                `json:"site,omitempty"`
	InactiveFlag       bool                `json:"inactiveFlag,omitempty"`
	Title              string              `json:"title,omitempty,omitempty"`
	DefaultPhoneType   string              `json:"defaultPhoneType,omitempty"`
	DefaultPhoneNbr    string              `json:"defaultPhoneNbr,omitempty"`
	DefaultBillingFlag bool                `json:"defaultBillingFlag,omitempty"`
	DefaultFlag        bool                `json:"defaultFlag,omitempty"`
	CompanyLocation    CompanyLocation     `json:"companyLocation,omitempty"`
	CommunicationItems []CommunicationItem `json:"communicationItems,omitempty"`
	Types              []Type              `json:"types,omitempty"`
}

type Ticket struct {
	Id                         int             `json:"id,omitempty"`
	Summary                    string          `json:"summary,omitempty"`
	InitialDescription         string          `json:"initialDescription,omitempty"`
	InitialInternalAnalysis    string          `json:"InitialInternalAnalysis,omitempty"`
	RecordType                 string          `json:"recordType,omitempty"`
	Board                      Board           `json:"board,omitempty"`
	Status                     Status          `json:"status,omitempty"`
	Company                    Company         `json:"company,omitempty"`
	Site                       Site            `json:"site,omitempty"`
	SiteName                   string          `json:"siteName,omitempty"`
	AddressLine1               string          `json:"addressLine1,omitempty"`
	AddressLine2               string          `json:"addressLine2,omitempty"`
	City                       string          `json:"city,omitempty"`
	Contact                    Contact         `json:"contact,omitempty"`
	ContactName                string          `json:"contactName,omitempty"`
	ContactPhoneNumber         string          `json:"contactPhoneNumber,omitempty"`
	ContactEmailAddress        string          `json:"contactEmailAddress,omitempty"`
	Type                       Type            `json:"type,omitempty"`
	Team                       Team            `json:"team,omitempty"`
	Owner                      Owner           `json:"owner,omitempty"`
	Priority                   Priority        `json:"priority,omitempty"`
	ServiceLocation            ServiceLocation `json:"serviceLocation,omitempty"`
	Source                     Source          `json:"source,omitempty"`
	RequiredDate               time.Time       `json:"requiredDate,omitempty"`
	AgreementType              string          `json:"agreementType,omitempty"`
	Severity                   string          `json:"severity,omitempty"`
	Impact                     string          `json:"impact,omitempty"`
	AllowAllClientsPortalView  bool            `json:"allowAllClientsPortalView,omitempty"`
	CustomerUpdatedFlag        bool            `json:"customerUpdatedFlag,omitempty"`
	AutomaticEmailContactFlag  bool            `json:"automaticEmailContactFlag,omitempty"`
	AutomaticEmailResourceFlag bool            `json:"automaticEmailResourceFlag,omitempty"`
	AutomaticEmailCcFlag       bool            `json:"automaticEmailCcFlag,omitempty"`
	ClosedFlag                 bool            `json:"closedFlag,omitempty"`
	ActualHours                float64         `json:"actualHours,omitempty"`
	Approved                   bool            `json:"approved,omitempty"`
	EstimatedExpenseCost       float64         `json:"estimatedExpenseCost,omitempty"`
	EstimatedExpenseRevenue    float64         `json:"estimatedExpenseRevenue,omitempty"`
	EstimatedProductCost       float64         `json:"estimatedProductCost,omitempty"`
	EstimatedProductRevenue    float64         `json:"estimatedProductRevenue,omitempty"`
	EstimatedTimeCost          float64         `json:"estimatedTimeCost,omitempty"`
	EstimatedTimeRevenue       float64         `json:"estimatedTimeRevenue,omitempty"`
	BillingMethod              string          `json:"billingMethod,omitempty"`
	SubBillingMethod           string          `json:"subBillingMethod,omitempty"`
	ResolveMinutes             int             `json:"resolveMinutes,omitempty"`
	ResPlanMinutes             int             `json:"resPlanMinutes,omitempty"`
	RespondMinutes             int             `json:"respondMinutes,omitempty"`
	IsInSla                    bool            `json:"isInSla,omitempty"`
	HasChildTicket             bool            `json:"hasChildTicket,omitempty"`
	HasMergedChildTicketFlag   bool            `json:"hasMergedChildTicketFlag,omitempty"`
	BillTime                   string          `json:"billTime,omitempty"`
	BillExpenses               string          `json:"billExpenses,omitempty"`
	BillProducts               string          `json:"billProducts,omitempty"`
	Location                   Location        `json:"location,omitempty"`
	Department                 Department      `json:"department,omitempty"`
	EscalationLevel            int             `json:"escalationLevel,omitempty"`
	CustomFields               []CustomField   `json:"customFields,omitempty"`
}

type Board struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type CommunicationItem struct {
	Id                int                   `json:"id,omitempty"`
	Type              CommunicationItemType `json:"type,omitempty"`
	Value             string                `json:"value,omitempty"`
	DefaultFlag       bool                  `json:"defaultFlag,omitempty"`
	Domain            string                `json:"domain,omitempty"`
	CommunicationType string                `json:"communicationType,omitempty"`
}

type CommunicationItemType struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type CompanyLocation struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Country struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type CustomField struct {
	Id               int    `json:"id,omitempty"`
	Caption          string `json:"caption,omitempty"`
	Type             string `json:"type,omitempty"`
	EntryMethod      string `json:"entryMethod,omitempty"`
	NumberOfDecimals int    `json:"numberOfDecimals,omitempty"`
	ConnectWiseId    string `json:"connectWiseId,omitempty"`
}

type DefaultContact struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Department struct {
	Id         int    `json:"id,omitempty"`
	Identifier string `json:"identifier,omitempty"`
	Name       string `json:"name,omitempty"`
}

type Location struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Owner struct {
	Id         int    `json:"id,omitempty"`
	Identifier string `json:"identifier,omitempty"`
	Name       string `json:"name,omitempty"`
}

type Priority struct {
	Id    int    `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Sort  int    `json:"sort,omitempty"`
	Level string `json:"level,omitempty"`
}

type ServiceLocation struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Site struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Source struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Status struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Team struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Territory struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Type struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Member struct {
	Id                  int    `json:"id"`
	Identifier          string `json:"identifier"`
	DisableOnlineFlag   bool   `json:"disableOnlineFlag"`
	LicenseClass        string `json:"licenseClass"`
	EnableMobileGpsFlag bool   `json:"enableMobileGpsFlag"`
	InactiveFlag        bool   `json:"inactiveFlag"`
	ClientId            string `json:"clientId"`
	FirstName           string `json:"firstName"`
	OfficeEmail         string `json:"officeEmail,omitempty"`
	DefaultEmail        string `json:"defaultEmail"`
	PrimaryEmail        string `json:"primaryEmail,omitempty"`
}

type TicketNote struct {
	Id                    int       `json:"id"`
	TicketId              int       `json:"ticketId"`
	Text                  string    `json:"text"`
	DetailDescriptionFlag bool      `json:"detailDescriptionFlag"`
	InternalAnalysisFlag  bool      `json:"internalAnalysisFlag"`
	ResolutionFlag        bool      `json:"resolutionFlag"`
	IssueFlag             bool      `json:"issueFlag"`
	Contact               Contact   `json:"contact,omitempty"`
	DateCreated           time.Time `json:"dateCreated"`
	CreatedBy             string    `json:"createdBy"`
	InternalFlag          bool      `json:"internalFlag"`
	ExternalFlag          bool      `json:"externalFlag"`
	Member                Member    `json:"member,omitempty"`
}
