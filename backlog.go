package okanoworld

import(
	"appengine"
	"appengine/urlfetch"
	"net/http"
	"strings"
	"io"
	"fmt"
	"encoding/xml"
	"encoding/json"
	"log"
)

/**
 * Backlog API を呼び出すためのクラス
 * @class
 * @param {string} space Backlogスペース名
 * @param {string} id ログインID
 * @param {string} pass ログインパスワード
 */
type Backlog struct {
	context appengine.Context
	space string
	id string
	pass string
	request *http.Request
}

/**
 * Backlogオブジェクトのファクトリメソッド
 * @function
 * @param {appengine.Context} c コンテキスト
 * @param {string} space Backlogスペース名
 * @param {stirng} id ログインid
 * @param {string} pass ログインパスワード
 * @returns {*Backlog} Backlogオブジェクト
 */
func newBacklog(c appengine.Context, space string, id string, pass string, request *http.Request) *Backlog {
	log.Println("newBacklog")
	var backlog = new(Backlog)
	backlog.context = c
	backlog.space = space
	backlog.id = id
	backlog.pass = pass
	backlog.request = request
	return backlog
}

/**
 * Backlog API 呼び出しの入り口
 * メソッド名やパラメータを含めてリクエストを投げる
 * http://okanoworld.appengine.com/backlog?method=xxxxxx&param=xxxxxx
 * @function
 * @param {http.ResponseWriter} w 応答先
 * @param {*http.Request} r リクエスト
 */
func requestBacklog(w http.ResponseWriter, r *http.Request) {
	var c = appengine.NewContext(r)
	var space = r.FormValue("space")
	var id = r.FormValue("id")
	var pass = r.FormValue("pass")
	var method = r.FormValue("method")
	
	var backlog = newBacklog(c, space, id, pass, r)
	var result = backlog.exec(method)
	var resultJSON []byte
	var err error
	resultJSON, err = json.Marshal(result)
	check(c, err)
	_, err = fmt.Fprintf(w, "%s", resultJSON)
	check(c, err)
}

/**
 * メソッドを実行して結果を返す
 * 無効なメソッド名が指定されている場合は何もせず nil を返す
 * 有効なメソッド名が指定されている場合は適切なメソッドへ投げる
 * @method
 * @memberof Backlog
 * @param {string} method 実行するメソッド名
 * Backlog API のメソッド名をキャメルケースにした文字列
 * @returns {[]map[string]string}
 */
func (this *Backlog) exec(method string) interface{} {
	if this.id == "" || this.space == "" || this.pass == "" || method == "" {
		return nil
	}
	
	var result []map[string]string
	log.Println("method check")
	switch method {
	case "get_projects":
		result = this.getProjects()
	case "find_issue":
		result = this.findIssue()
	case "get_issue_types":
		result = this.getIssueTypes()
	case "get_components":
		result = this.getComponents()
	case "get_statuses":
		result = this.getStatuses()
	default:
		log.Println("default")
		return nil
	}

	return result
}

/**
 * XML を backlog のサーバへ渡すための Reader
 * @class
 */
type XMLReader struct {
	io.Reader
	data []byte
}

/**
 * xml を読み込む
 * @method
 * @memberof XMLReader
 */
func (this *XMLReader) Read(p []byte) (int, error) {
	var i int
	var n int
	var err error
	
	for i = 0; i < len(p) - 1; i++ {
		if i > len(this.data) - 1 {
			break
		}
		p[i] = this.data[i]
	}
	
	if i == len(p) - 1 {
		n = len(p)
		err = nil
	} else {
		n = len(this.data)
		err = io.EOF
	}
	return n, err
}

/**
 * Backlog API へのリクエスト作成・送信・受信までを行う
 * @method
 * @memberof Backlog
 * @param {[]byte}  送信するXML
 * @returns {[]byte} 受信したXML
 */
func (this *Backlog) sendXML(xml []byte) []byte {
	var err error
	
	// 送信先URL作成
	var url string
	url = strings.Join([]string{"https://", this.space, ".backlog.jp/XML-RPC"}, "")
	
	// リクエストXMLを作成
	var xmlReader *XMLReader
	xmlReader = new(XMLReader)
	xmlReader.data = xml
	
	// HTTPリクエスト作成
	var client *http.Client
	var request *http.Request
	client = urlfetch.Client(this.context)
	request, err = http.NewRequest("POST", url, xmlReader)
	check(this.context, err)
	request.SetBasicAuth(this.id, this.pass)
	request.Header.Set("Content-Type", "text/xml")
	
	// HTTPリクエスト送信と受信
	var response *http.Response
	response, err = client.Do(request)
	check(this.context, err)
	
	// HTTPレスポンスを読み出す
	var responseXML []byte
	responseXML = make([]byte, response.ContentLength)
	_, err = response.Body.Read(responseXML)
	check(this.context, err)
	
	return responseXML
}

/**
 * backlog.getProjects() を実行する
 * @method
 * @memberof Backlog
 * @returns {[]map[string]stromg} プロジェクトリスト
 * @see http://www.backlog.jp/api/method1_1.html
 */
func (this *Backlog) getProjects() []map[string]string {
	var err error
	
	// XMLの作成
	var requestXML string
	requestXML = `
		<?xml version="1.0" encoding="utf-8"?>
		<methodCall>
			<methodName>backlog.getProjects</methodName>
			<params />
		</methodCall>
	`
	requestXML = this.serialize(requestXML)
	
	// XMLの送信と受信
	var responseXML []byte
	responseXML = this.sendXML([]byte(requestXML));
	
	// レスポンスXMLをデコード
	type ValueXML struct {
		Chardata string `xml:",chardata"`
		I4 string `xml:"i4"`
	}
	type MemberXML struct {
		Name string `xml:"name"`
		Value ValueXML `xml:"value"`
	}
	type ProjectXML struct {
		Members []MemberXML `xml:"struct>member"`
	}
	type ResponseXML struct {
		Projects []ProjectXML `xml:"params>param>value>array>data>value"`
	}
	var result = new(ResponseXML)
	err = xml.Unmarshal(responseXML, result)
	check(this.context, err)
	
	// 結果を返す
	var projects []map[string]string
	var i int
	var j int
	var projectXML ProjectXML
	var memberXML MemberXML
	projects = make([]map[string]string, len(result.Projects))
	for i = 0; i < len(result.Projects); i++ {
		projects[i] = make(map[string]string, 3)
		projectXML = result.Projects[i]
		for j = 0; j < len(projectXML.Members); j++ {
			memberXML = projectXML.Members[j]
			switch memberXML.Name {
			case "name":
				projects[i]["name"] = memberXML.Value.Chardata
			case "key":
				projects[i]["key"] = memberXML.Value.Chardata
			case "url":
				projects[i]["url"] = memberXML.Value.Chardata
			case "id":
				projects[i]["id"] = memberXML.Value.I4
			}
		}
	}
	return projects
}

/**
 * backlog.findIssue を実行する
 * @method
 * @memberof Backlog
 * @returns {[]map[string]string} タスクリスト
 */
func (this *Backlog) findIssue() []map[string]string {
	var projectId string
	var err error
	projectId = this.request.FormValue("project")
	
	// XMLの作成
	var requestXML string
	requestXML = `
		<?xml version="1.0" encoding="utf-8"?>
		<methodCall>
			<methodName>backlog.findIssue</methodName>
			<params>
				<param>
					<value>
						<struct>
							<member>
								<name>projectId</name>
								<value>
									<int>[PROJECT_ID]</int>
								</value>
							</member>
							<member>
								<name>statusId</name>
								<value>
									<array>
										<data>
											<value>
												<int>1</int>
											</value>
											<value>
												<int>2</int>
											</value>
										</data>
									</array>
								</value>
							</member>
						</struct>
					</value>
				</param>
			</params>
		</methodCall>
	`
	requestXML = strings.Replace(requestXML, "[PROJECT_ID]", projectId, 1)
	requestXML = this.serialize(requestXML)
	

	// XMLの送信と受信
	var responseBytes []byte
	responseBytes = this.sendXML([]byte(requestXML));
		
	// レスポンスXMLを解析
	type SubValueXML struct {
		Chardata string `xml:",chardata"`
		Name string `xml:"name"`
	}
	type SubMemberXML struct {
		Name string `xml:"name"`
		Value SubValueXML `xml:"value"`
	}
	type SubStructXML struct {
		Members []SubMemberXML `xml:"member"`
	}
	type ValueXML struct {
		Chardata string `xml:",chardata"`
		I4 string `xml:"i4"`
		Name string `xml:"name"`
		Array SubStructXML `xml:"array>data>value>struct"`
		Struct SubStructXML `xml:"struct"`
	}
	type MemberXML struct {
		Raw string `xml:",innerxml"`
		Name string `xml:"name"`
		Value ValueXML `xml:"value"`
	}
	type StructXML struct {
		Members []MemberXML `xml:"member"`
	}
	type ResponseXML struct {
		Structs []StructXML `xml:"params>param>value>array>data>value>struct"`
	}
	var responseXML *ResponseXML
	responseXML = new(ResponseXML)
	err = xml.Unmarshal(responseBytes, responseXML)
	check(this.context, err)
	
	// 解析したXMLから必要なデータを抽出する
	var i int
	var j int
	var structXML StructXML
	var memberXML MemberXML
	var result []map[string]string
	result = make([]map[string]string, len(responseXML.Structs))
	for i = 0; i < len(responseXML.Structs); i++ {
		structXML = responseXML.Structs[i]
		result[i] = make(map[string]string)
		for j = 0; j < len(structXML.Members); j++ {
			memberXML = structXML.Members[j]
			switch memberXML.Name {
			case "key":
				result[i]["key"] = memberXML.Value.Chardata
			case "url":
				result[i]["url"] = memberXML.Value.Chardata
			case "summary":
				result[i]["summary"] = memberXML.Value.Chardata
			case "created_on":
				result[i]["created_on"] = memberXML.Value.Chardata
			case "components":
				var k int
				var subStructXML SubStructXML
				var subMemberXML SubMemberXML
				subStructXML = memberXML.Value.Array
				for k = 0; k < len(subStructXML.Members); k++ {
					subMemberXML = subStructXML.Members[k]
					switch(subMemberXML.Name) {
					case "name":
						result[i]["components"] = subMemberXML.Value.Chardata
					}
				}
			case "status":
				var k int
				var subStructXML SubStructXML
				var subMemberXML SubMemberXML
				subStructXML = memberXML.Value.Struct
				for k = 0; k < len(subStructXML.Members); k++ {
					subMemberXML = subStructXML.Members[k]
					switch(subMemberXML.Name) {
					case "name":
						result[i]["status"] = subMemberXML.Value.Chardata
					}
				}
			case "assigner":
				var k int
				var subStructXML SubStructXML
				var subMemberXML SubMemberXML
				subStructXML = memberXML.Value.Struct
				for k = 0; k < len(subStructXML.Members); k++ {
					subMemberXML = subStructXML.Members[k]
					switch(subMemberXML.Name) {
					case "name":
						result[i]["assigner"] = subMemberXML.Value.Chardata
					}
				}
			case "description":
				result[i]["description"] = memberXML.Value.Chardata
			}
		}
	}
	
	return result
}

/**
 * 種別の取得
 * @method
 * @memberof Backlog
 */
func (this *Backlog) getIssueTypes() []map[string]string {
	var err error
	var projectId string
	projectId = this.request.FormValue("project")
	
	// リクエストXMLの作成
	var requestXML string
	requestXML = `
		<?xml version="1.0" encoding="utf-8"?>
		<methodCall>
		  <methodName>backlog.getIssueTypes</methodName>
		  <params>
			<param>
			  <value>
				<int>[PROJECT_ID]</int>
			  </value>
			</param>
		  </params>
		</methodCall>
	`
	requestXML = strings.Replace(requestXML, "[PROJECT_ID]", projectId, 1)
	requestXML = this.serialize(requestXML)
	
	// XMLの送信と受信
	var responseXML []byte
	responseXML = this.sendXML([]byte(requestXML));
	
	// 結果の解析
	type SubValue struct {
		Chardata string `xml:",chardata"`
		I4 string `xml:"i4"`
	}
	type Member struct {
		Name string `xml:"name"`
		Value SubValue `xml:"value"`
	}
	type Value struct {
		Members []Member `xml:"struct>member"`
	}
	type MethodResponse struct {
		Values []Value `xml:"params>param>value>array>data>value"`
	}
	var methodResponse *MethodResponse
	methodResponse = new(MethodResponse)
	err = xml.Unmarshal(responseXML, methodResponse)
	check(this.context, err)
	
	var result []map[string]string
	var i, j int
	var value Value
	var member Member
	result = make([]map[string]string, len(methodResponse.Values))
	for i = 0; i < len(methodResponse.Values); i++ {
		result[i] = make(map[string]string, 2)
		value = methodResponse.Values[i]
		for j = 0; j < len(value.Members); j++ {
			member = value.Members[j]
			switch member.Name {
			case "id":
				result[i]["id"] = member.Value.I4
			case "name":
				result[i]["name"] = member.Value.Chardata
			}
		}
	}
	
	return result
}

/**
 * カテゴリの取得
 * @method
 * @memberof Backlog
 * @returns {[]map[string]string} カテゴリ一覧
 */
func (this *Backlog) getComponents() []map[string]string {
	var projectId string
	projectId = this.request.FormValue("project")
	
	var requestXML = `
	<?xml version="1.0" encoding="utf-8"?>
	<methodCall>
	  <methodName>backlog.getComponents</methodName>
	  <params>
		<param>
		  <value>
			<int>[PROJECT_ID]</int>
		  </value>
		</param>
	  </params>
	</methodCall>
	`
	requestXML = strings.Replace(requestXML, "[PROJECT_ID]", projectId, 1)
	requestXML = this.serialize(requestXML)

	var responseXML []byte
	responseXML = this.sendXML([]byte(requestXML));
	
	type SubValue struct {
		Chardata string `xml:",chardata"`
		I4 string `xml:"i4"`
	}
	type Member struct {
		Name string `xml:"name"`
		Value SubValue `xml:"value"`
	}
	type Value struct {
		Members []Member `xml:"struct>member"`
	}
	type MethodResponse struct {
		Values []Value `xml:"params>param>value>array>data>value"`
	}
	
	var methodResponse *MethodResponse
	var err error
	methodResponse = new(MethodResponse)
	err = xml.Unmarshal(responseXML, methodResponse)
	check(this.context, err)
	
	var i, j int
	var value Value
	var member Member
	var result []map[string]string
	result = make([]map[string]string, len(methodResponse.Values))
	for i = 0; i < len(methodResponse.Values); i++ {
		value = methodResponse.Values[i]
		result[i] = make(map[string]string, len(value.Members))
		for j = 0; j < len(value.Members); j++ {
			member = value.Members[j]
			switch member.Name {
			case "id":
				result[i]["id"] = member.Value.I4
			case "name":
				result[i]["name"] = member.Value.Chardata
			}
		}
	}
	
	return result
}

/**
 * 状態の取得
 * @method
 * @memberof Backlog
 * @returns {[]map[string]string} 状態リスト
 */
func (this *Backlog) getStatuses() []map[string]string {
	var requestXML string
	requestXML = `
		<?xml version="1.0" encoding="utf-8"?>
		<methodCall>
		  <methodName>backlog.getStatuses</methodName>
		  <params />
		</methodCall>
	`
	requestXML = this.serialize(requestXML)
	
	var responseXML []byte
	responseXML = this.sendXML([]byte(requestXML))
	
	type SubValue struct {
		Chardata string `xml:",chardata"`
		I4 string `xml:"i4"`
	}
	type Member struct {
		Name string `xml:"name"`
		Value SubValue `xml:"value"`
	}
	type Value struct {
		Members []Member `xml:"struct>member"`
	}
	type MethodResponse struct {
		Values []Value `xml:"params>param>value>array>data>value"`
	}
	
	var err error
	var methodResponse *MethodResponse
	methodResponse = new(MethodResponse)	
	err = xml.Unmarshal(responseXML, methodResponse)
	check(this.context, err)
	
	var result []map[string]string
	result = make([]map[string]string, len(methodResponse.Values))
	var i, j int
	var value Value
	var member Member
	for i = 0; i < len(methodResponse.Values); i++ {
		value = methodResponse.Values[i]
		result[i] = make(map[string]string, 2)
		for j = 0; j < len(value.Members); j++ {
			member = value.Members[j]
			switch member.Name {
			case "id":
				result[i]["id"] = member.Value.I4
			case "name":
				result[i]["name"] = member.Value.Chardata
			}
		}
	}
	
	return result
}

/**
 * 文字列の改行とタブを削除する
 * @param {string} str 変換する文字列
 * @param {string} 直列化された文字列
 */
func (this *Backlog) serialize(str string) string {
	str = strings.Replace(str, "\n", "", -1)
	str = strings.Replace(str, "\t", "", -1)
	return str
}