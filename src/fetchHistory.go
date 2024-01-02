package src

import (
	"fmt"
	"github.com/deanishe/awgo"
	"os"
	"strings"
)

var FetchHistory = func(wf *aw.Workflow, query string) {
	titleQuery, domainQuery, isDomainSearch := ParseUserQuery(query)

	// 2022/9/7, blaketang, 添加url模糊搜索
	// 2022/9/7, blaketang, 禁止掉域名搜索，域名判断太绝对了
	// 2023-12-30, 恢复域名搜索
	// isDomainSearch = false
	var dbQuery = ""
	if len(domainQuery) > 0 {
		dbQuery = fmt.Sprintf(`
		SELECT urls.id, urls.title, urls.url, urls.last_visit_time FROM urls
		WHERE urls.url LIKE '%%%s%%' and 
		%s
		ORDER BY last_visit_time DESC
	`, domainQuery,
			combineLikeCondition("urls.title", titleQuery))
	} else {
		dbQuery = fmt.Sprintf(`
		SELECT urls.id, urls.title, urls.url, urls.last_visit_time FROM urls
		WHERE %s
		ORDER BY last_visit_time DESC
	`, combineLikeCondition("urls.title", titleQuery))
	}

	fmt.Fprintln(os.Stderr, `查询SQL: `, dbQuery)

	historyDB := GetHistoryDB(wf)

	rows, err := historyDB.Query(dbQuery)
	CheckError(err)

	var urlTitle string
	var url string
	var urlId string
	var urlLastVisitTime int64

	var itemCount = 0
	var previousTitle = ""

	for rows.Next() {
		if itemCount >= Conf.ResultCountLimit {
			break
		}

		err := rows.Scan(&urlId, &urlTitle, &url, &urlLastVisitTime)
		CheckError(err)

		if previousTitle == urlTitle {
			continue
		}

		domainName := ExtractDomainName(url)
		if isDomainSearch && !strings.Contains(domainName, domainQuery) {
			continue
		}

		unixTimestamp := ConvertChromeTimeToUnixTimestamp(urlLastVisitTime)
		localeTimeStr := GetLocaleString(unixTimestamp)

		item := wf.NewItem(urlTitle).
			Subtitle(localeTimeStr).
			Valid(true).
			Quicklook(url).
			Autocomplete(urlTitle).
			Arg(url).
			Copytext(url).
			Largetype(urlTitle)

		item.Cmd().Subtitle("Press Enter to copy this url to clipboard")

		iconPath := fmt.Sprintf(`%s/%s.png`, GetFaviconDirectoryPath(wf), domainName)

		if FileExist(iconPath) {
			item.Icon(&aw.Icon{iconPath, ""})
		}

		previousTitle = urlTitle
		itemCount += 1
	}
}
