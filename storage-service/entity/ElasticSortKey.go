package entity

// ElasticSortKey maps sort field names to their last-result values.
// Used as a pagination cursor for Elasticsearch search_after.
type ElasticSortKey []string
