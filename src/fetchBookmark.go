package src

import (
	"fmt"
	"github.com/deanishe/awgo"
	"github.com/mozillazg/go-pinyin"
	"sort"
	"strings"
	"unicode"
)

var FetchBookmark = func(wf *aw.Workflow, query string) {
	InitBookmarkJsonTraversal()
	bookmarkRoot := GetChromeBookmark()
	input, flags := ParseQueryFlags(query)
	input, domainQuery, isDomainSearch := ParseUserQuery(input)
	var bookmarks []BookmarkItem

	if folderId, ok := flags["folderId"]; ok {
		folders := TraverseBookmarkJSONObject(bookmarkRoot, TraverseBookmarkJsonOption{Targets: []string{"folder"}, Depth: 99})

		for _, folder := range folders {
			if folder.Id == folderId {
				bookmarks = TraverseBookmarkArray(folder.Children, TraverseBookmarkJsonOption{Targets: []string{"url"}, Depth: 1})
			}
		}
		if bookmarks == nil {
			panic(fmt.Sprintf("folderId not found: %s", folderId))
		}
	} else {
		bookmarks = TraverseBookmarkJSONObject(bookmarkRoot, TraverseBookmarkJsonOption{Targets: []string{"url"}, Depth: 99})
	}

	//2023-12-30 域名搜索(其实是url)
	if isDomainSearch {
		var new_bookmarks []BookmarkItem
		// 全部转换成小写
		domainQuery = strings.ToLower(domainQuery)
		for i := 0; i < len(bookmarks); i++ {
			if strings.Contains(strings.ToLower(bookmarks[i].Url), domainQuery) {
				new_bookmarks = append(new_bookmarks, bookmarks[i])
			}
		}
		bookmarks = new_bookmarks
	}

	historyDB := GetHistoryDB(wf)
	visitHistories, err := historyDB.Query("SELECT url FROM urls")
	CheckError(err)

	visitFrequency := make(map[string]int)

	for visitHistories.Next() {
		var url string
		err := visitHistories.Scan(&url)
		CheckError(err)

		visitFrequency[url] += 1
	}

	sort.Slice(bookmarks, func(i, j int) bool {
		ithFreq := visitFrequency[bookmarks[i].Url]
		jthFreq := visitFrequency[bookmarks[j].Url]

		if ithFreq > 0 && jthFreq > 0 {
			if ithFreq > jthFreq {
				return true
			} else {
				return false
			}
		}

		if ithFreq > 0 {
			return true
		}

		return false
	})

	for _, bookmark := range bookmarks {
		domainName := ExtractDomainName(bookmark.Url)
		iconPath := fmt.Sprintf(`%s/%s.png`, GetFaviconDirectoryPath(wf), domainName)
		CheckError(err)

		item := wf.NewItem(bookmark.Name).
			Valid(true).
			Subtitle(bookmark.Url).
			Quicklook(bookmark.Url).
			Arg(bookmark.Url).
			Copytext(bookmark.Url).
			Autocomplete(bookmark.Name).
			Largetype(bookmark.Name).
			Match(toPinYin(bookmark.Name))

		item.Cmd().Subtitle("Press Enter to copy this url to clipboard")

		if FileExist(iconPath) {
			item.Icon(&aw.Icon{iconPath, ""})
		}
	}

	// 修正搜索不支持空格分隔的问题，同时也支持顺序调换
	for _, word := range strings.Split(input, " ") {
		wf.Filter(word)
	}
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func toPinYin(s string) string {
	pinYinArgs := pinyin.NewArgs()
	// 是否多音字
	// 必须开启，否则后续通过 slice[0] == "" 的判断会失效，因为
	// 非多单字的情况下，每个字符的数组，只会返回1个元素
	pinYinArgs.Heteronym = true
	pinYinArgs.Fallback = func(r rune, a pinyin.Args) []string {
		// "" 作为一个标记
		return []string{"", string(r)}
	}
	pinyin_str := ""
	for _, slice := range pinyin.Pinyin(s, pinYinArgs) {
		if slice[0] == "" {
			pinyin_str += slice[1]
			continue
		}
		// 只取第 1 个拼音，其它直接不要
		pinyin_str += strings.ToUpper(slice[0][0:1]) + slice[0][1:]

		//line := ""
		//for _, str := range slice {
		//	// 防止重复
		//	if strings.Contains(line, str) {
		//		continue
		//	}
		//	line += str
		//	pinyin_str += strings.ToUpper(str[0:1]) + str[1:]
		//}
	}
	return s + pinyin_str
}
