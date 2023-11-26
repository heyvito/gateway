package gateway

func unique[E comparable, S interface{ ~[]E }](in S) S {
	items := make(map[E]bool, len(in))
	for _, v := range in {
		items[v] = true
	}
	out := make(S, 0, len(items))
	for v := range items {
		out = append(out, v)
	}
	return out
}
