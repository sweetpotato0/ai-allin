package reranker

import "context"

type queryContextKey struct{}

// ContextWithQuery stores the raw query string so rerankers that rely on text (not just vectors)
// can retrieve it without changing the Rank signature.
func ContextWithQuery(ctx context.Context, query string) context.Context {
	return context.WithValue(ctx, queryContextKey{}, query)
}

// QueryFromContext extracts the query string previously stored with ContextWithQuery.
func QueryFromContext(ctx context.Context) (string, bool) {
	val := ctx.Value(queryContextKey{})
	if val == nil {
		return "", false
	}
	query, ok := val.(string)
	return query, ok
}
