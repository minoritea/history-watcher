package main

// zsh only utils

func unmetafy(s string) string {
	var (
		ret []byte
		i   int
	)
	bs := []byte(s)
	l := len(bs)
	for {
		if i >= l {
			break
		}
		if bs[i] == 0x83 {
			i++
			if i >= l {
				break
			}
			bs[i] ^= 0x20
		}
		ret = append(ret, bs[i])
		i++
	}
	return string(ret)
}
