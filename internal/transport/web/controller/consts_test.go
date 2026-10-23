package controller

// Shared fixtures used across the controller tests.
const (
	testArticleID = "hash123"
	testUserID    = "user456"

	testHashID1         = "hash1"
	testHashID2         = "hash2"
	testSimilarHashID1  = "similar1"
	testSimilarHashID2  = "similar2"
	testTitle           = "Test"
	testTitle1          = "Article 1"
	testTitle2          = "Article 2"
	testArticleTitle    = "Test Article"
	testSimilarTitle1   = "Similar Article 1"
	testSimilarTitle2   = "Similar Article 2"
	testSourceLessWrong = "lesswrong"

	testMaxAgeCacheControl = "max-age=3600"
	testSearchBody         = `{"text": "test"}`
	testInvalidValue       = "invalid"

	testNoCacheCaseName    = "no_cache_for_authenticated_user"
	testFetchErrorCaseName = "fetch_error"

	// muxVarArticleID is the gorilla/mux route variable holding the article ID.
	muxVarArticleID = "article_id"
)
