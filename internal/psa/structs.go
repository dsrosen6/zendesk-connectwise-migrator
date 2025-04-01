package psa

type Company struct {
	Id          int    `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	DeletedFlag bool   `json:"deletedFlag,omitempty"`
}

type ContactPostBody struct {
	FirstName          string              `json:"firstName,omitempty"`
	LastName           string              `json:"lastName,omitempty"`
	Company            Company             `json:"company,omitempty"`
	CommunicationItems []CommunicationItem `json:"communicationItems,omitempty"`
}

type Contact struct {
	Id int `json:"id,omitempty"`
}

type Ticket struct {
	Id                      int           `json:"id,omitempty"`
	Summary                 string        `json:"summary,omitempty"`
	InitialInternalAnalysis string        `json:"InitialInternalAnalysis,omitempty"`
	Board                   *Board        `json:"board,omitempty"`
	Status                  *Status       `json:"status,omitempty"`
	Company                 *Company      `json:"company,omitempty"`
	Contact                 *Contact      `json:"contact,omitempty"`
	Owner                   *Owner        `json:"owner,omitempty"`
	CustomFields            []CustomField `json:"customFields,omitempty"`
}

type Board struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type CommunicationItem struct {
	Type              CommunicationItemType `json:"type,omitempty"`
	Value             string                `json:"value,omitempty"`
	CommunicationType string                `json:"communicationType,omitempty"`
}

type CommunicationItemType struct {
	Name string `json:"name,omitempty"`
}

type CustomField struct {
	Id    int `json:"id,omitempty"`
	Value any `json:"value"`
}

type Owner struct {
	Id         int    `json:"id,omitempty"`
	Identifier string `json:"identifier,omitempty"`
}

type Status struct {
	Id   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type BoardType struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Member struct {
	Id           int    `json:"id,omitempty"`
	PrimaryEmail string `json:"primaryEmail,omitempty"`
}

type TicketNote struct {
	Text                  string   `json:"text,omitempty"`
	DetailDescriptionFlag bool     `json:"detailDescriptionFlag,omitempty"`
	InternalAnalysisFlag  bool     `json:"internalAnalysisFlag,omitempty"`
	Contact               *Contact `json:"contact,omitempty"`
	Member                *Member  `json:"member,omitempty"`
}
