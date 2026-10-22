package webhook

import (
	"fmt"
	"sync"
	"testing"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

// Fails under -race before GetOrCompile compiled against a snapshot of preexistingExpressions.
func TestExpressionCacheConcurrentAddAndCompile(t *testing.T) {
	t.Parallel()
	c := NewExpressionCache()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		cond := admissionregistrationv1.MatchCondition{
			Name:       fmt.Sprintf("cond-%d", i),
			Expression: fmt.Sprintf("object.metadata.name != 'x%d'", i),
		}
		wg.Add(2)
		go func() {
			defer wg.Done()
			c.AddExpression(cond)
		}()
		go func() {
			defer wg.Done()
			c.GetOrCompile(admissionregistrationv1.MatchCondition{Name: cond.Name + "b", Expression: cond.Expression})
		}()
	}
	wg.Wait()
}
