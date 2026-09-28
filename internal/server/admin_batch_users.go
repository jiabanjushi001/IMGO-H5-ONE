package server

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

const maxBatchUserCount = 100

type batchUserInput struct {
	Count      int
	Prefix     string
	Password   string
	InviteCode string
	CustomerID int64
}

type batchCustomer struct {
	UserID  int64
	Account string
}

type batchTaskPayload struct {
	Input batchUserInput `json:"input"`
	Scope adminScope     `json:"scope"`
	IP    string         `json:"ip"`
}

type batchTaskPlan struct {
	InviterID     int64
	Customers     []batchCustomer
	CustomerState M
	Welcome       string
	AccountNumber int
}

func batchUserInputFromParams(params M) (batchUserInput, error) {
	input := batchUserInput{
		Count:      int(number(params["count"])),
		Prefix:     strings.TrimSpace(str(params["account_prefix"])),
		Password:   str(params["password"]),
		InviteCode: strings.TrimSpace(str(params["parent_invite_code"])),
		CustomerID: number(params["customer_user_id"]),
	}
	if input.Count < 1 || input.Count > maxBatchUserCount {
		return batchUserInput{}, clientError{"账号个数须为 1–100", 400}
	}
	if len(input.Password) < 6 || len(input.Password) > 30 {
		return batchUserInput{}, clientError{"统一密码长度须为 6–30 字节", 400}
	}
	if len(input.Prefix) > 20 {
		return batchUserInput{}, clientError{"账号前缀最多 20 个字符", 400}
	}
	for _, ch := range input.Prefix {
		if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_' || ch == '-' {
			continue
		}
		return batchUserInput{}, clientError{"账号前缀只能包含字母、数字、下划线和短横线", 400}
	}
	if input.CustomerID < 0 {
		return batchUserInput{}, clientError{"客服账号无效", 400}
	}
	return input, nil
}

const batchAccountAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

func secureRandomIndex(limit int) (int, error) {
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(limit)))
	if err != nil {
		return 0, err
	}
	return int(value.Int64()), nil
}

func randomBatchAccount(prefix string) (string, error) {
	if prefix == "" {
		prefix = "u"
	}
	var suffix strings.Builder
	for i := 0; i < 8; i++ {
		index, err := secureRandomIndex(len(batchAccountAlphabet))
		if err != nil {
			return "", err
		}
		suffix.WriteByte(batchAccountAlphabet[index])
	}
	return prefix + suffix.String(), nil
}

func sequentialBatchAccount(prefix string, sequence int) string {
	if sequence < 1 {
		sequence = 1
	}
	return fmt.Sprintf("%s%03d", prefix, sequence)
}

func (a *App) nextBatchAccountNumber(ctx context.Context, db DB, prefix string) (int, error) {
	if prefix == "" {
		return 0, nil
	}
	accounts, err := rows(ctx, db, "SELECT account FROM "+a.t("user")+" WHERE LEFT(account,?)=?", len(prefix), prefix)
	if err != nil {
		return 0, err
	}
	next := 1
	for _, item := range accounts {
		account := str(item["account"])
		if len(account) <= len(prefix) || !strings.EqualFold(account[:len(prefix)], prefix) {
			continue
		}
		suffix := account[len(prefix):]
		if len(suffix) < 3 {
			continue
		}
		value, parseErr := strconv.Atoi(suffix)
		if parseErr == nil && value >= next {
			next = value + 1
		}
	}
	return next, nil
}

var batchSurnames = []rune("赵钱孙李周吴郑王冯陈褚卫蒋沈韩杨朱秦尤许何吕施张孔曹严华金魏陶姜戚谢邹喻柏水窦章云苏潘葛奚范彭郎鲁韦昌马苗凤花方俞任袁柳鲍史唐费廉岑薛雷贺倪汤滕殷罗毕郝邬安常乐于时傅皮卞齐康伍余元卜顾孟平黄穆萧尹姚邵汪祁毛禹狄米贝明臧计伏成戴宋茅庞熊纪舒屈项祝董梁杜阮蓝闵席季麻强贾路娄危江童颜郭梅盛林钟徐邱骆高夏蔡田樊胡凌霍虞万支柯管卢莫经房裘缪干解应宗丁宣贲邓郁单杭洪包诸左石崔吉龚程嵇邢滑裴陆荣翁荀羊於惠甄曲家封芮羿储靳汲邴糜松井段富巫乌焦巴弓牧隗山谷车侯宓蓬全郗班仰秋仲伊宫宁仇栾暴甘钭厉戎祖武符刘景詹束龙叶幸司韶郜黎蓟薄印宿白怀蒲台从鄂索咸籍赖卓蔺屠蒙池乔阴胥能苍双闻莘党翟谭贡劳逄姬申扶堵冉宰郦雍璩桑桂濮牛寿通边扈燕冀浦尚农温别庄晏柴瞿阎连茹习艾鱼容向古易慎戈廖庾终暨居衡步都耿满弘匡国文寇广禄阙东欧沃利蔚越隆师巩厍聂晁勾敖融冷辛阚那简饶空曾毋沙乜养鞠须丰巢关蒯相查后荆红游竺权逯盖益桓公")
var batchGivenNames = []rune("子文嘉欣宇轩浩然雨桐梓涵思远若曦一诺安然明哲俊杰静怡雅琪诗涵晨曦佳宁睿泽可馨依琳书瑶亦辰梦琪昊天心妍奕然芷晴景行知夏清越锦程星河云舒舒然以安乐彤楚悦沐阳天佑语嫣婉清皓月嘉言卓然启航博雅思齐瑞雪")

func randomChineseName() (string, error) {
	surnameIndex, err := secureRandomIndex(len(batchSurnames))
	if err != nil {
		return "", err
	}
	givenCountIndex, err := secureRandomIndex(3)
	if err != nil {
		return "", err
	}
	name := []rune{batchSurnames[surnameIndex]}
	for i := 0; i < givenCountIndex+1; i++ {
		index, err := secureRandomIndex(len(batchGivenNames))
		if err != nil {
			return "", err
		}
		name = append(name, batchGivenNames[index])
	}
	return string(name), nil
}

func randomBatchSex() (int64, error) {
	index, err := secureRandomIndex(2)
	return int64(index), err
}

func (a *App) batchMemberInviter(r *request, db DB, scope adminScope) (int64, error) {
	return a.batchMemberInviterFor(r.ctx(), db, r.s("parent_invite_code"), scope)
}

func (a *App) batchMemberInviterFor(ctx context.Context, db DB, inviteCode string, scope adminScope) (int64, error) {
	code := strings.TrimSpace(inviteCode)
	if code == "" {
		if scope.Global {
			return 0, nil
		}
		return 0, clientError{"导师批量创建账号必须填写上级邀请码", 400}
	}
	inviterID, err := a.resolveInviter(ctx, db, code)
	if err != nil {
		return 0, err
	}
	if scope.Global || inviterID == scope.AgentUserID {
		return inviterID, nil
	}
	if err = a.requireScopedUser(ctx, db, scope, inviterID); err != nil {
		var denied clientError
		if errors.As(err, &denied) && denied.code == 403 {
			return 0, clientError{"上级邀请码只能选择本人或自己团队的成员", 400}
		}
		return 0, err
	}
	return inviterID, nil
}

func (a *App) batchCustomers(ctx context.Context, db DB, userIDs []int64) ([]batchCustomer, error) {
	customers := make([]batchCustomer, 0, len(userIDs))
	for _, userID := range userIDs {
		row, err := one(ctx, db, "SELECT user_id,account FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", userID)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		customers = append(customers, batchCustomer{UserID: userID, Account: str(row["account"])})
	}
	return customers, nil
}

func (a *App) batchMentorCustomers(ctx context.Context, tx *sql.Tx, agentID int64) ([]batchCustomer, M, string, error) {
	chat, err := a.registrationChatConfig(ctx, tx)
	if err != nil {
		return nil, nil, "", err
	}
	autoUser, _, inherited, _, err := a.agentSetting(ctx, tx, agentID)
	if err != nil {
		return nil, nil, "", err
	}
	if inherited {
		autoUser = obj(chat["autoAddUser"])
	}
	if number(autoUser["status"]) != 1 {
		return nil, nil, "", clientError{"请先在导师设置中启用并配置默认客服", 400}
	}
	customerIDs, err := agentCustomerIDs(autoUser)
	if err != nil {
		return nil, nil, "", err
	}
	customers, err := a.batchCustomers(ctx, tx, customerIDs)
	if err != nil {
		return nil, nil, "", err
	}
	if len(customers) == 0 {
		return nil, nil, "", clientError{"导师默认客服不可用，请先更新导师设置", 400}
	}
	state, err := a.loadAgentAutoState(ctx, tx, agentID)
	if err != nil {
		return nil, nil, "", err
	}
	return customers, state, str(autoUser["welcome"]), nil
}

func (a *App) assignBatchCustomer(ctx context.Context, tx *sql.Tx, userID int64, customer batchCustomer, welcome string) error {
	if customer.UserID < 1 {
		return nil
	}
	if err := update(ctx, tx, a.t("user"), M{"cs_uid": customer.UserID}, "user_id=?", userID); err != nil {
		return err
	}
	now := time.Now().Unix()
	if err := a.addMutualFriendship(ctx, tx, customer.UserID, userID, now); err != nil {
		return err
	}
	if welcome == "" {
		return nil
	}
	content, _ := encryptContent(a.cfg.ChatKey, sanitizeText(welcome))
	_, err := insert(ctx, tx, a.t("message"), M{"id": randomID()[:32], "from_user": customer.UserID, "to_user": userID, "chat_identify": chatKey(customer.UserID, userID, 0), "content": content, "type": "text", "create_time": now, "is_group": 0, "is_read": 0, "is_last": 1, "status": 1})
	return err
}

func duplicateKeyError(err error) bool {
	var duplicate *mysql.MySQLError
	return errors.As(err, &duplicate) && duplicate.Number == 1062
}

func (a *App) createBatchUser(ctx context.Context, tx *sql.Tx, input batchUserInput, accountNumber *int, ip string) (int64, string, string, int64, error) {
	for attempt := 0; attempt < 1000; attempt++ {
		var account string
		var err error
		if input.Prefix == "" {
			account, err = randomBatchAccount("")
		} else {
			account = sequentialBatchAccount(input.Prefix, *accountNumber)
			*accountNumber = *accountNumber + 1
		}
		if err != nil {
			return 0, "", "", 0, err
		}
		if len(account) > 32 {
			return 0, "", "", 0, clientError{"账号前缀生成的编号超过账号长度限制", 400}
		}
		name, err := randomChineseName()
		if err != nil {
			return 0, "", "", 0, err
		}
		sex, err := randomBatchSex()
		if err != nil {
			return 0, "", "", 0, err
		}
		userID, err := a.createUser(ctx, tx, M{"account": account, "realname": name, "password": input.Password}, ip)
		if duplicateKeyError(err) {
			continue
		}
		if err != nil {
			return 0, "", "", 0, err
		}
		if err = update(ctx, tx, a.t("user"), M{"sex": sex}, "user_id=?", userID); err != nil {
			return 0, "", "", 0, err
		}
		return userID, account, name, sex, nil
	}
	return 0, "", "", 0, clientError{"账号生成冲突，请重试", 409}
}

func (a *App) batchTaskKey() string {
	if a.cfg.ChatKey != "" {
		return a.cfg.ChatKey
	}
	return a.cfg.JWTKey
}

func (a *App) encodeBatchTaskPayload(payload batchTaskPayload) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return encryptContent(a.batchTaskKey(), string(raw))
}

func (a *App) decodeBatchTaskPayload(ciphertext string) (batchTaskPayload, error) {
	plain, err := decryptContent(a.batchTaskKey(), ciphertext)
	if err != nil {
		return batchTaskPayload{}, err
	}
	var payload batchTaskPayload
	if err = json.Unmarshal([]byte(plain), &payload); err != nil {
		return batchTaskPayload{}, err
	}
	return payload, nil
}

func (a *App) encodeBatchTaskResults(items []M) (string, error) {
	raw, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	return encryptContent(a.batchTaskKey(), string(raw))
}

func (a *App) decodeBatchTaskResults(ciphertext string) ([]M, error) {
	if ciphertext == "" {
		return []M{}, nil
	}
	plain, err := decryptContent(a.batchTaskKey(), ciphertext)
	if err != nil {
		return nil, err
	}
	items := []M{}
	if err = json.Unmarshal([]byte(plain), &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (a *App) prepareBatchTask(ctx context.Context, tx *sql.Tx, input batchUserInput, scope adminScope) (batchTaskPlan, error) {
	inviterID, err := a.batchMemberInviterFor(ctx, tx, input.InviteCode, scope)
	if err != nil {
		return batchTaskPlan{}, err
	}
	accountNumber, err := a.nextBatchAccountNumber(ctx, tx, input.Prefix)
	if err != nil {
		return batchTaskPlan{}, err
	}
	plan := batchTaskPlan{InviterID: inviterID, CustomerState: M{}, AccountNumber: accountNumber}
	if scope.Global {
		if input.CustomerID < 1 {
			return plan, nil
		}
		plan.Customers, err = a.batchCustomers(ctx, tx, []int64{input.CustomerID})
		if err == nil && len(plan.Customers) == 0 {
			err = clientError{"指定的客服账号不存在或已停用", 400}
		}
		return plan, err
	}
	plan.Customers, plan.CustomerState, plan.Welcome, err = a.batchMentorCustomers(ctx, tx, scope.AgentUserID)
	return plan, err
}

func batchTaskCustomer(plan *batchTaskPlan, global bool) batchCustomer {
	if len(plan.Customers) == 0 {
		return batchCustomer{}
	}
	if len(plan.Customers) == 1 {
		if !global {
			plan.CustomerState["user_id"] = plan.Customers[0].UserID
		}
		return plan.Customers[0]
	}
	ids := make([]int64, len(plan.Customers))
	for i := range plan.Customers {
		ids[i] = plan.Customers[i].UserID
	}
	nextID := nextCustomerService(ids, number(plan.CustomerState["user_id"]))
	for _, candidate := range plan.Customers {
		if candidate.UserID == nextID {
			plan.CustomerState["user_id"] = candidate.UserID
			return candidate
		}
	}
	return batchCustomer{}
}

func (a *App) batchTaskView(ctx context.Context, actorID int64, taskID string) (M, error) {
	query := "SELECT task_id,status,total_count,completed_count,success_count,failed_count,result_cipher,error_message,created_at,updated_at,finished_at FROM " + a.t("imgo_batch_user_task") + " WHERE actor_user_id=?"
	args := []any{actorID}
	if taskID != "" {
		query += " AND task_id=? LIMIT 1"
		args = append(args, taskID)
	} else {
		query += " AND status IN ('queued','running') ORDER BY created_at DESC LIMIT 1"
	}
	row, err := one(ctx, a.db, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return M{"found": false}, nil
	}
	if err != nil {
		return nil, err
	}
	items, err := a.decodeBatchTaskResults(str(row["result_cipher"]))
	if err != nil {
		return nil, err
	}
	total := number(row["total_count"])
	completed := number(row["completed_count"])
	percent := int64(0)
	if total > 0 {
		percent = completed * 100 / total
	}
	if str(row["status"]) == "completed" {
		percent = 100
	}
	return M{
		"found": true, "task_id": str(row["task_id"]), "status": str(row["status"]),
		"total_count": total, "completed_count": completed, "success_count": number(row["success_count"]),
		"failed_count": number(row["failed_count"]), "percent": percent, "items": items,
		"error_message": str(row["error_message"]), "created_at": number(row["created_at"]),
		"updated_at": number(row["updated_at"]), "finished_at": number(row["finished_at"]),
	}, nil
}

func (a *App) batchUserTaskStatus(r *request) (any, error) {
	r.c.Header("Cache-Control", "no-store")
	taskID := strings.TrimSpace(r.s("task_id"))
	if taskID != "" && (len(taskID) != 32 || strings.Trim(taskID, "0123456789abcdef") != "") {
		return nil, r.fail("任务编号无效")
	}
	return a.batchTaskView(r.ctx(), r.uid(), taskID)
}

func (a *App) saveBatchTaskState(ctx context.Context, db DB, taskID, status string, total, completed, failed int, items []M, message string) error {
	ciphertext, err := a.encodeBatchTaskResults(items)
	if err != nil {
		return err
	}
	finishedAt := int64(0)
	if status == "completed" || status == "failed" {
		finishedAt = time.Now().Unix()
		_, err = db.ExecContext(ctx, "UPDATE "+a.t("imgo_batch_user_task")+" SET status=?,completed_count=?,success_count=?,failed_count=?,result_cipher=?,error_message=?,updated_at=?,finished_at=?,active_actor_user_id=NULL WHERE task_id=?", status, completed, len(items), failed, ciphertext, message, time.Now().Unix(), finishedAt, taskID)
		return err
	}
	_, err = db.ExecContext(ctx, "UPDATE "+a.t("imgo_batch_user_task")+" SET status=?,completed_count=?,success_count=?,failed_count=?,result_cipher=?,error_message=?,updated_at=?,finished_at=? WHERE task_id=?", status, completed, len(items), failed, ciphertext, message, time.Now().Unix(), finishedAt, taskID)
	return err
}

func (a *App) failBatchTask(ctx context.Context, taskID string, total int, items []M, cause error) {
	message := "批量创建失败"
	if cause != nil && cause.Error() != "" {
		message = cause.Error()
	}
	if len(message) > 500 {
		message = message[:500]
	}
	if err := a.saveBatchTaskState(ctx, a.db, taskID, "failed", total, len(items), 1, items, message); err != nil {
		a.log.Error("save failed batch user task", "task_id", taskID, "error", err)
	}
}

func (a *App) runBatchUserTask(ctx context.Context, taskID string) {
	row, err := one(ctx, a.db, "SELECT status,total_count,request_cipher,result_cipher FROM "+a.t("imgo_batch_user_task")+" WHERE task_id=?", taskID)
	if err != nil || (str(row["status"]) != "queued" && str(row["status"]) != "running") {
		if err != nil && !errors.Is(err, context.Canceled) {
			a.log.Error("load batch user task", "task_id", taskID, "error", err)
		}
		return
	}
	payload, err := a.decodeBatchTaskPayload(str(row["request_cipher"]))
	if err != nil {
		a.failBatchTask(ctx, taskID, int(number(row["total_count"])), nil, err)
		return
	}
	if !payload.Scope.Global {
		payload.Scope.referralTable = a.t("imgo_referral_path")
	}
	items, err := a.decodeBatchTaskResults(str(row["result_cipher"]))
	if err != nil {
		a.failBatchTask(ctx, taskID, payload.Input.Count, nil, err)
		return
	}
	if _, err = a.db.ExecContext(ctx, "UPDATE "+a.t("imgo_batch_user_task")+" SET status='running',updated_at=? WHERE task_id=? AND status IN ('queued','running')", time.Now().Unix(), taskID); err != nil {
		return
	}
	prep, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		a.failBatchTask(ctx, taskID, payload.Input.Count, items, err)
		return
	}
	plan, err := a.prepareBatchTask(ctx, prep, payload.Input, payload.Scope)
	_ = prep.Rollback()
	if err != nil {
		a.failBatchTask(ctx, taskID, payload.Input.Count, items, err)
		return
	}
	for len(items) < payload.Input.Count {
		if err = ctx.Err(); err != nil {
			return
		}
		customer := batchTaskCustomer(&plan, payload.Scope.Global)
		tx, txErr := a.db.BeginTx(ctx, nil)
		if txErr != nil {
			a.failBatchTask(ctx, taskID, payload.Input.Count, items, txErr)
			return
		}
		userID, account, name, sex, createErr := a.createBatchUser(ctx, tx, payload.Input, &plan.AccountNumber, payload.IP)
		if createErr == nil && plan.InviterID > 0 {
			createErr = a.bindInviter(ctx, tx, userID, plan.InviterID)
		}
		if createErr == nil {
			createErr = a.assignBatchCustomer(ctx, tx, userID, customer, plan.Welcome)
		}
		if createErr == nil && !payload.Scope.Global {
			createErr = a.saveAgentAutoState(ctx, tx, payload.Scope.AgentUserID, plan.CustomerState)
		}
		sexLabel := "女"
		if sex == 1 {
			sexLabel = "男"
		}
		if createErr == nil {
			items = append(items, M{"user_id": userID, "account": account, "password": payload.Input.Password, "realname": name, "sex": sex, "sex_label": sexLabel, "parent_invite_code": payload.Input.InviteCode, "customer_user_id": customer.UserID, "customer_account": customer.Account})
			createErr = a.saveBatchTaskState(ctx, tx, taskID, "running", payload.Input.Count, len(items), 0, items, "")
		}
		if createErr == nil {
			createErr = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
		if createErr != nil {
			a.failBatchTask(ctx, taskID, payload.Input.Count, items, createErr)
			return
		}
	}
	if err = a.saveBatchTaskState(ctx, a.db, taskID, "completed", payload.Input.Count, len(items), 0, items, ""); err != nil && !errors.Is(err, context.Canceled) {
		a.log.Error("complete batch user task", "task_id", taskID, "error", err)
	}
}

func (a *App) enqueueBatchUserTask(taskID string) {
	a.batchTaskMu.Lock()
	if a.batchTaskContext == nil {
		a.batchTaskContext, a.batchTaskCancel = context.WithCancel(context.Background())
	}
	if a.batchTasks == nil {
		a.batchTasks = map[string]struct{}{}
	}
	if _, exists := a.batchTasks[taskID]; exists {
		a.batchTaskMu.Unlock()
		return
	}
	a.batchTasks[taskID] = struct{}{}
	ctx := a.batchTaskContext
	a.wg.Add(1)
	a.batchTaskMu.Unlock()
	go func() {
		defer a.wg.Done()
		defer func() {
			a.batchTaskMu.Lock()
			delete(a.batchTasks, taskID)
			a.batchTaskMu.Unlock()
		}()
		a.runBatchUserTask(ctx, taskID)
	}()
}

func (a *App) StartBatchUserTasks() {
	a.batchTaskMu.Lock()
	if a.batchTaskContext == nil {
		a.batchTaskContext, a.batchTaskCancel = context.WithCancel(context.Background())
	}
	a.batchTaskMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tasks, err := rows(ctx, a.db, "SELECT task_id FROM "+a.t("imgo_batch_user_task")+" WHERE status IN ('queued','running') ORDER BY created_at")
	if err != nil {
		a.log.Error("recover batch user tasks", "error", err)
		return
	}
	for _, task := range tasks {
		a.enqueueBatchUserTask(str(task["task_id"]))
	}
}

func (a *App) batchAddUsers(r *request, scope adminScope) (any, error) {
	input, err := batchUserInputFromParams(r.p)
	if err != nil {
		return nil, err
	}
	validation, err := a.db.BeginTx(r.ctx(), nil)
	if err != nil {
		return nil, err
	}
	if _, err = a.prepareBatchTask(r.ctx(), validation, input, scope); err != nil {
		_ = validation.Rollback()
		return nil, err
	}
	_ = validation.Rollback()
	payload := batchTaskPayload{Input: input, Scope: adminScope{Global: scope.Global, AgentUserID: scope.AgentUserID}, IP: a.clientIP(r.c)}
	ciphertext, err := a.encodeBatchTaskPayload(payload)
	if err != nil {
		return nil, err
	}
	taskID := randomID()[:32]
	now := time.Now().Unix()
	_, err = a.db.ExecContext(r.ctx(), "INSERT INTO "+a.t("imgo_batch_user_task")+" (task_id,actor_user_id,active_actor_user_id,status,total_count,request_cipher,result_cipher,created_at,updated_at) VALUES (?,?,?,'queued',?,?,'',?,?)", taskID, r.uid(), r.uid(), input.Count, ciphertext, now, now)
	if duplicateKeyError(err) {
		view, viewErr := a.batchTaskView(r.ctx(), r.uid(), "")
		if viewErr == nil {
			view["resumed"] = true
		}
		return view, viewErr
	}
	if err != nil {
		return nil, err
	}
	a.enqueueBatchUserTask(taskID)
	return M{"found": true, "task_id": taskID, "status": "queued", "total_count": input.Count, "completed_count": 0, "success_count": 0, "failed_count": 0, "percent": 0, "items": []M{}}, nil
}
