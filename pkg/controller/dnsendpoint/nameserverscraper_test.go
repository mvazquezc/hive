package dnsendpoint

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/hive/pkg/controller/dnsendpoint/nameserver/mock"
)

func TestGetEndpoint(t *testing.T) {
	rootDomain := "domain.com"
	domain := "test.domain.com"
	cases := []struct {
		name             string
		nameServers      rootDomainsMap
		expectRootDomain bool
		expectedValues   sets.String
	}{
		{
			name: "empty",
		},
		{
			name: "no root domain",
			nameServers: rootDomainsMap{
				"other-domain": nameServersMap{},
			},
		},
		{
			name: "empty root domain",
			nameServers: rootDomainsMap{
				"domain.com": nameServersMap{},
			},
			expectRootDomain: true,
		},
		{
			name: "no domain",
			nameServers: rootDomainsMap{
				rootDomain: nameServersMap{
					"other.domain.com": endpointState{},
				},
			},
			expectRootDomain: true,
		},
		{
			name: "single namespace value",
			nameServers: rootDomainsMap{
				rootDomain: nameServersMap{
					domain: endpointState{
						nsValues: sets.NewString("test-value"),
					},
				},
			},
			expectRootDomain: true,
			expectedValues:   sets.NewString("test-value"),
		},
		{
			name: "multiple namespace values",
			nameServers: rootDomainsMap{
				rootDomain: nameServersMap{
					domain: endpointState{
						nsValues: sets.NewString("test-value-1", "test-value-2", "test-value-3"),
					},
				},
			},
			expectRootDomain: true,
			expectedValues:   sets.NewString("test-value-1", "test-value-2", "test-value-3"),
		},
		{
			name: "many root domains and domains",
			nameServers: rootDomainsMap{
				rootDomain: nameServersMap{
					domain: endpointState{
						nsValues: sets.NewString("test-value"),
					},
					"other.domain.com": endpointState{
						nsValues: sets.NewString("other-value"),
					},
				},
				"other-domain": nameServersMap{
					"sub-domain.other-domain": endpointState{
						nsValues: sets.NewString("another-value"),
					},
				},
			},
			expectRootDomain: true,
			expectedValues:   sets.NewString("test-value"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cut := &nameServerScraper{nameServers: tc.nameServers}
			actualRootDomain, actualValues := cut.GetEndpoint("test.domain.com")
			if tc.expectRootDomain {
				assert.Equal(t, "domain.com", actualRootDomain, "unexpected root domain")
			} else {
				assert.Empty(t, actualRootDomain, "expected to not find root domain")
			}
			assert.Equal(t, tc.expectedValues, actualValues, "unexpected name server values")
		})
	}
}

func TestAddEndpoint(t *testing.T) {
	rootDomain := "domain.com"
	domain := "test.domain.com"
	objectKey := client.ObjectKey{Namespace: "test-namespace", Name: "test-name"}
	values := sets.NewString("test-value-1", "test-value-2", "test-value-3")
	cases := []struct {
		name                string
		nameServers         rootDomainsMap
		expectedNameServers rootDomainsMap
	}{
		{
			name: "empty",
		},
		{
			name: "no root domain",
			nameServers: rootDomainsMap{
				"other-domain": nameServersMap{},
			},
			expectedNameServers: rootDomainsMap{
				"other-domain": nameServersMap{},
			},
		},
		{
			name: "empty root domain",
			nameServers: rootDomainsMap{
				rootDomain: nameServersMap{},
			},
			expectedNameServers: rootDomainsMap{
				rootDomain: nameServersMap{
					domain: endpointState{
						objectKey: objectKey,
						nsValues:  values,
					},
				},
			},
		},
		{
			name: "no domain",
			nameServers: rootDomainsMap{
				rootDomain: nameServersMap{
					"other.domain.com": endpointState{},
				},
			},
			expectedNameServers: rootDomainsMap{
				rootDomain: nameServersMap{
					"other.domain.com": endpointState{},
					domain: endpointState{
						objectKey: objectKey,
						nsValues:  values,
					},
				},
			},
		},
		{
			name: "update domain",
			nameServers: rootDomainsMap{
				rootDomain: nameServersMap{
					domain: endpointState{
						objectKey: client.ObjectKey{Namespace: "other-namespace", Name: "other-name"},
						nsValues:  sets.NewString("other-value"),
					},
				},
			},
			expectedNameServers: rootDomainsMap{
				rootDomain: nameServersMap{
					domain: endpointState{
						objectKey: objectKey,
						nsValues:  values,
					},
				},
			},
		},
		{
			name: "multiple root domains",
			nameServers: rootDomainsMap{
				rootDomain: nameServersMap{
					"other.domain.com": endpointState{},
				},
				"other-domain": nameServersMap{
					"subdomain.other-domain": endpointState{},
				},
			},
			expectedNameServers: rootDomainsMap{
				rootDomain: nameServersMap{
					"other.domain.com": endpointState{},
					domain: endpointState{
						objectKey: objectKey,
						nsValues:  values,
					},
				},
				"other-domain": nameServersMap{
					"subdomain.other-domain": endpointState{},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cut := &nameServerScraper{nameServers: tc.nameServers}
			cut.AddEndpoint(objectKey, domain, values)
			assert.Equal(t, tc.expectedNameServers, cut.nameServers, "unexpected changes to name servers")
		})
	}
}

func TestRemoveEndpoint(t *testing.T) {
	rootDomain := "domain.com"
	domain := "test.domain.com"
	cases := []struct {
		name                string
		nameServers         rootDomainsMap
		expectedNameServers rootDomainsMap
	}{
		{
			name: "empty",
		},
		{
			name: "no root domain",
			nameServers: rootDomainsMap{
				"other-domain": nameServersMap{},
			},
			expectedNameServers: rootDomainsMap{
				"other-domain": nameServersMap{},
			},
		},
		{
			name: "empty root domain",
			nameServers: rootDomainsMap{
				rootDomain: nameServersMap{},
			},
			expectedNameServers: rootDomainsMap{
				rootDomain: nameServersMap{},
			},
		},
		{
			name: "no domain",
			nameServers: rootDomainsMap{
				rootDomain: nameServersMap{
					"other.domain.com": endpointState{},
				},
			},
			expectedNameServers: rootDomainsMap{
				rootDomain: nameServersMap{
					"other.domain.com": endpointState{},
				},
			},
		},
		{
			name: "remove domain",
			nameServers: rootDomainsMap{
				rootDomain: nameServersMap{
					domain: endpointState{
						objectKey: client.ObjectKey{Namespace: "test-namespace", Name: "test-name"},
						nsValues:  sets.NewString("test-value"),
					},
				},
			},
			expectedNameServers: rootDomainsMap{
				rootDomain: nameServersMap{},
			},
		},
		{
			name: "multiple root domains",
			nameServers: rootDomainsMap{
				rootDomain: nameServersMap{
					"other.domain.com": endpointState{},
					domain: endpointState{
						objectKey: client.ObjectKey{Namespace: "test-namespace", Name: "test-name"},
						nsValues:  sets.NewString("test-value"),
					},
				},
				"other-domain": nameServersMap{
					"subdomain.other-domain": endpointState{},
				},
			},
			expectedNameServers: rootDomainsMap{
				rootDomain: nameServersMap{
					"other.domain.com": endpointState{},
				},
				"other-domain": nameServersMap{
					"subdomain.other-domain": endpointState{},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cut := &nameServerScraper{nameServers: tc.nameServers}
			cut.RemoveEndpoint(domain)
			assert.Equal(t, tc.expectedNameServers, cut.nameServers, "unexpected changes to name servers")
		})
	}
}

func TestScrape(t *testing.T) {
	cases := []struct {
		name                string
		testDuration        time.Duration
		scrapePeriod        time.Duration
		rootDomains         []string
		nameServers         rootDomainsMap
		configureQuery      func(mockQuery *mock.MockQuery)
		expectedNameServers rootDomainsMap
		expectedChanges     []client.ObjectKey
	}{
		{
			name:        "single root domain",
			rootDomains: []string{"domain.com"},
			configureQuery: func(mockQuery *mock.MockQuery) {
				mockQuery.EXPECT().Get("domain.com").Return(nil, nil)
			},
		},
		{
			name:        "multiple root domains",
			rootDomains: []string{"domain1.com", "domain2.com", "domain3.com"},
			configureQuery: func(mockQuery *mock.MockQuery) {
				mockQuery.EXPECT().Get("domain1.com").Return(nil, nil)
				mockQuery.EXPECT().Get("domain2.com").Return(nil, nil)
				mockQuery.EXPECT().Get("domain3.com").Return(nil, nil)
			},
		},
		{
			name:         "scrape after scrape period",
			scrapePeriod: 2 * time.Second,
			rootDomains:  []string{"domain1.com", "domain2.com"},
			configureQuery: func(mockQuery *mock.MockQuery) {
				mockQuery.EXPECT().Get("domain1.com").Return(nil, nil).Times(2)
				mockQuery.EXPECT().Get("domain2.com").Return(nil, nil).Times(2)
			},
		},
		{
			name:         "scrape after error",
			testDuration: 12 * time.Second,
			rootDomains:  []string{"domain1.com", "domain2.com"},
			configureQuery: func(mockQuery *mock.MockQuery) {
				mockQuery.EXPECT().Get("domain1.com").Return(nil, errors.New("test error"))
				mockQuery.EXPECT().Get("domain1.com").Return(nil, nil)
				mockQuery.EXPECT().Get("domain2.com").Return(nil, nil)
			},
		},
		{
			name:        "untracked domain",
			rootDomains: []string{"domain.com"},
			configureQuery: func(mockQuery *mock.MockQuery) {
				mockQuery.EXPECT().Get("domain.com").
					Return(
						map[string]sets.String{
							"test.domain.com": sets.NewString("test-value"),
						},
						nil,
					)
			},
		},
		{
			name:        "update domain",
			rootDomains: []string{"domain.com"},
			nameServers: rootDomainsMap{
				"domain.com": nameServersMap{
					"test.domain.com": endpointState{
						objectKey: client.ObjectKey{Namespace: "test-namespace", Name: "test-name"},
						nsValues:  sets.NewString("old-value"),
					},
				},
			},
			configureQuery: func(mockQuery *mock.MockQuery) {
				mockQuery.EXPECT().Get("domain.com").
					Return(
						map[string]sets.String{
							"test.domain.com": sets.NewString("test-value"),
						},
						nil,
					)
			},
			expectedNameServers: rootDomainsMap{
				"domain.com": nameServersMap{
					"test.domain.com": endpointState{
						objectKey: client.ObjectKey{Namespace: "test-namespace", Name: "test-name"},
						nsValues:  sets.NewString("test-value"),
					},
				},
			},
			expectedChanges: []client.ObjectKey{{Namespace: "test-namespace", Name: "test-name"}},
		},
		{
			name:        "no changes to domain",
			rootDomains: []string{"domain.com"},
			nameServers: rootDomainsMap{
				"domain.com": nameServersMap{
					"test.domain.com": endpointState{
						objectKey: client.ObjectKey{Namespace: "test-namespace", Name: "test-name"},
						nsValues:  sets.NewString("test-value"),
					},
				},
			},
			configureQuery: func(mockQuery *mock.MockQuery) {
				mockQuery.EXPECT().Get("domain.com").
					Return(
						map[string]sets.String{
							"test.domain.com": sets.NewString("test-value"),
						},
						nil,
					)
			},
			expectedNameServers: rootDomainsMap{
				"domain.com": nameServersMap{
					"test.domain.com": endpointState{
						objectKey: client.ObjectKey{Namespace: "test-namespace", Name: "test-name"},
						nsValues:  sets.NewString("test-value"),
					},
				},
			},
		},
		{
			name:        "update multiple domains",
			rootDomains: []string{"domain.com"},
			nameServers: rootDomainsMap{
				"domain.com": nameServersMap{
					"changed-1.domain.com": endpointState{
						objectKey: client.ObjectKey{Namespace: "test-namespace", Name: "test-changed-1"},
						nsValues:  sets.NewString("old-value-1"),
					},
					"changed-2.domain.com": endpointState{
						objectKey: client.ObjectKey{Namespace: "test-namespace", Name: "test-changed-2"},
						nsValues:  sets.NewString("old-value-2"),
					},
					"changed-3.domain.com": endpointState{
						objectKey: client.ObjectKey{Namespace: "test-namespace", Name: "test-changed-3"},
						nsValues:  sets.NewString("old-value-3a", "old-value-3b"),
					},
					"unchanged.domain.com": endpointState{
						objectKey: client.ObjectKey{Namespace: "test-namespace", Name: "test-unchanged"},
						nsValues:  sets.NewString("test-value-4"),
					},
				},
			},
			configureQuery: func(mockQuery *mock.MockQuery) {
				mockQuery.EXPECT().Get("domain.com").
					Return(
						map[string]sets.String{
							"changed-1.domain.com": sets.NewString("test-value-1"),
							"changed-2.domain.com": sets.NewString("test-value-2a", "test-value-2b"),
							"changed-3.domain.com": sets.NewString("test-value-3"),
							"unchanged.domain.com": sets.NewString("test-value-4"),
							"untacked.domain.com":  sets.NewString("test-value-5"),
						},
						nil,
					)
			},
			expectedNameServers: rootDomainsMap{
				"domain.com": nameServersMap{
					"changed-1.domain.com": endpointState{
						objectKey: client.ObjectKey{Namespace: "test-namespace", Name: "test-changed-1"},
						nsValues:  sets.NewString("test-value-1"),
					},
					"changed-2.domain.com": endpointState{
						objectKey: client.ObjectKey{Namespace: "test-namespace", Name: "test-changed-2"},
						nsValues:  sets.NewString("test-value-2a", "test-value-2b"),
					},
					"changed-3.domain.com": endpointState{
						objectKey: client.ObjectKey{Namespace: "test-namespace", Name: "test-changed-3"},
						nsValues:  sets.NewString("test-value-3"),
					},
					"unchanged.domain.com": endpointState{
						objectKey: client.ObjectKey{Namespace: "test-namespace", Name: "test-unchanged"},
						nsValues:  sets.NewString("test-value-4"),
					},
				},
			},
			expectedChanges: []client.ObjectKey{
				{Namespace: "test-namespace", Name: "test-changed-1"},
				{Namespace: "test-namespace", Name: "test-changed-2"},
				{Namespace: "test-namespace", Name: "test-changed-3"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockQuery := mock.NewMockQuery(mockCtrl)
			tc.configureQuery(mockQuery)
			changeNotifications := make(chan client.ObjectKey, 100)
			notifyChange := func(objectKey client.ObjectKey) {
				changeNotifications <- objectKey
			}
			cut := newNameServerScraper(log.StandardLogger(), mockQuery, tc.rootDomains, notifyChange)
			if tc.scrapePeriod > 0 {
				cut.scrapePeriod = tc.scrapePeriod
			}
			if tc.nameServers != nil {
				cut.nameServers = tc.nameServers
			}
			stop := make(chan struct{})
			go func() {
				sleepTime := tc.testDuration
				if sleepTime <= 0 {
					sleepTime = 3 * time.Second
				}
				time.Sleep(sleepTime)
				stop <- struct{}{}
			}()
			err := cut.Start(stop)
			assert.NoError(t, err, "unexpected error starting scraper")
			expectedNameServers := tc.expectedNameServers
			if len(expectedNameServers) == 0 {
				expectedNameServers = make(rootDomainsMap, len(tc.rootDomains))
				for _, d := range tc.rootDomains {
					expectedNameServers[d] = nameServersMap{}
				}
			}
			assert.Equal(t, expectedNameServers, cut.nameServers, "unexpected changes to name servers")
			actualChanges := []client.ObjectKey{}
			for {
				empty := false
				select {
				case key := <-changeNotifications:
					actualChanges = append(actualChanges, key)
				default:
					empty = true
				}
				if empty {
					break
				}
			}
			if len(tc.expectedChanges) == 0 {
				assert.Empty(t, actualChanges, "expected no change notifications")
			} else {
				assert.ElementsMatch(t, tc.expectedChanges, actualChanges, "unexpected change notifications")
			}
		})
	}
}
