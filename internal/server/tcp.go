package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	tcpProtocolVersion = 1
	tcpMaxMessageBytes = 1 << 20
	tcpIdleTimeout     = 90 * time.Second
	tcpWriteTimeout    = 15 * time.Second
	tcpRequestTimeout  = 30 * time.Second
)

type tcpRequest struct {
	Version int             `json:"version"`
	ID      string          `json:"id"`
	Action  string          `json:"action"`
	Data    json.RawMessage `json:"data"`
}

type tcpResponse struct {
	Version int    `json:"version"`
	ID      string `json:"id"`
	Code    int    `json:"code"`
	Message string `json:"msg"`
	Data    any    `json:"data,omitempty"`
}

type tcpConnectionState struct {
	authToken string
	deviceID  string
	peer      *tcpPeer
}

type tcpSessionBindInput struct {
	AuthToken string `json:"authToken"`
	DeviceID  string `json:"deviceId"`
}

func ListenTCP(addr, certFile, keyFile string) (net.Listener, error) {
	if strings.TrimSpace(addr) == "" {
		return nil, errors.New("IMGO_TCP_ADDR 未配置")
	}
	if (certFile == "") != (keyFile == "") {
		return nil, errors.New("IMGO_TCP_TLS_CERT 和 IMGO_TCP_TLS_KEY 必须同时配置")
	}
	if certFile == "" {
		return net.Listen("tcp", addr)
	}
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("加载 TCP TLS 证书失败: %w", err)
	}
	return tls.Listen("tcp", addr, &tls.Config{
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS12,
	})
}

// ServeTCP adds the native Android transport without replacing the existing
// HTTP routes or WebSocket hub. Both transports call the same App methods and
// therefore share users, sessions, configuration, and business data.
func (a *App) ServeTCP(ctx context.Context, listener net.Listener) error {
	if a.tcpHub == nil {
		a.tcpHub = newTCPHub(a)
	}
	serveCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-serveCtx.Done()
		_ = listener.Close()
	}()

	var connections sync.WaitGroup
	defer connections.Wait()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if serveCtx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("接受 TCP 连接失败: %w", err)
		}
		connections.Add(1)
		go func() {
			defer connections.Done()
			a.serveTCPConnection(serveCtx, conn)
		}()
	}
}

func (a *App) serveTCPConnection(ctx context.Context, conn net.Conn) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetKeepAlive(true)
		_ = tcpConn.SetNoDelay(true)
	}
	peer := a.tcpHub.add(conn)
	defer func() {
		uid := a.tcpHub.remove(peer)
		if uid > 0 && a.hub != nil && a.hub.online(uid) == 0 {
			a.hub.broadcast("isOnline", M{"id": uid, "is_online": 0})
		}
	}()
	go a.writeTCPPeer(peer)

	closed := make(chan struct{})
	defer close(closed)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-closed:
		}
	}()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 4096), tcpMaxMessageBytes)
	state := &tcpConnectionState{peer: peer}
	for {
		_ = conn.SetReadDeadline(time.Now().Add(tcpIdleTimeout))
		if !scanner.Scan() {
			return
		}

		requestCtx, cancel := context.WithTimeout(ctx, tcpRequestTimeout)
		response := a.handleTCPLine(requestCtx, state, conn.RemoteAddr().String(), scanner.Bytes())
		cancel()
		if !a.tcpHub.enqueue(peer, response) {
			return
		}
	}
}

func (a *App) writeTCPPeer(peer *tcpPeer) {
	writer := bufio.NewWriter(peer.conn)
	for {
		select {
		case <-peer.done:
			return
		case payload := <-peer.send:
			_ = peer.conn.SetWriteDeadline(time.Now().Add(tcpWriteTimeout))
			if _, err := writer.Write(payload); err != nil {
				a.log.Debug("write TCP payload", "error", err)
				peer.stop()
				return
			}
			if err := writer.WriteByte('\n'); err != nil {
				a.log.Debug("write TCP delimiter", "error", err)
				peer.stop()
				return
			}
			if err := writer.Flush(); err != nil {
				a.log.Debug("flush TCP payload", "error", err)
				peer.stop()
				return
			}
		}
	}
}

func (a *App) handleTCPLine(ctx context.Context, state *tcpConnectionState, remoteAddr string, line []byte) tcpResponse {
	var envelope tcpRequest
	if err := json.Unmarshal(line, &envelope); err != nil {
		return tcpFailure("", 400, "JSON 格式错误")
	}
	if envelope.Version != tcpProtocolVersion {
		return tcpFailure(envelope.ID, 400, "不支持的协议版本")
	}
	if strings.TrimSpace(envelope.ID) == "" {
		return tcpFailure("", 400, "请求 ID 不能为空")
	}

	data, message, err := a.handleTCPAction(ctx, state, remoteAddr, envelope)
	if err != nil {
		code, publicMessage := tcpPublicError(err)
		if code == 500 {
			a.log.Error("handle TCP request", "action", envelope.Action, "error", err)
		}
		return tcpFailure(envelope.ID, code, publicMessage)
	}
	return tcpResponse{Version: tcpProtocolVersion, ID: envelope.ID, Code: 0, Message: message, Data: data}
}

func (a *App) handleTCPAction(ctx context.Context, state *tcpConnectionState, remoteAddr string, envelope tcpRequest) (any, string, error) {
	switch envelope.Action {
	case "system.ping":
		return M{"type": "pong", "time": time.Now().UnixMilli()}, "成功", nil
	case "auth.login":
		params, err := tcpParams(envelope.Data)
		if err != nil {
			return nil, "", err
		}
		// Native member login must not turn itself into a backend-admin login or
		// try to bind a WebSocket client identifier supplied over another transport.
		delete(params, "admin_login")
		delete(params, "google_code")
		delete(params, "client_id")
		result, err := a.login(a.newTCPBusinessRequest(ctx, remoteAddr, "/common/pub/login", params))
		return result, "登录成功", err
	case "auth.register":
		params, err := tcpParams(envelope.Data)
		if err != nil {
			return nil, "", err
		}
		result, err := a.registerUser(a.newTCPBusinessRequest(ctx, remoteAddr, "/common/pub/register", params))
		if err != nil {
			return nil, "", err
		}
		uid := number(obj(result)["user_id"])
		user, err := one(ctx, a.db, "SELECT * FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", uid)
		if err != nil {
			return nil, "", err
		}
		return a.safeUser(user), "注册成功", nil
	case "auth.sendCode":
		params, err := tcpParams(envelope.Data)
		if err != nil {
			return nil, "", err
		}
		_, err = a.sendCode(a.newTCPBusinessRequest(ctx, remoteAddr, "/common/pub/sendCode", params))
		return M{"expiresIn": 300}, "验证码已发送", err
	case "session.bind":
		var input tcpSessionBindInput
		if err := json.Unmarshal(envelope.Data, &input); err != nil {
			return nil, "", clientError{"请求参数格式错误", 400}
		}
		input.AuthToken = strings.TrimSpace(input.AuthToken)
		input.DeviceID = strings.TrimSpace(input.DeviceID)
		if input.AuthToken == "" {
			return nil, "", clientError{"登录状态已失效", 401}
		}
		if input.DeviceID == "" {
			return nil, "", clientError{"设备 ID 不能为空", 400}
		}
		user, cl, err := a.authenticate(ctx, input.AuthToken)
		if err != nil {
			return nil, "", clientError{"登录状态已失效", 401}
		}
		if _, err = a.recordChatIP(ctx, number(user["user_id"]), remoteHost(remoteAddr)); err != nil {
			return nil, "", err
		}
		state.authToken = input.AuthToken
		state.deviceID = input.DeviceID
		if a.tcpHub != nil {
			if err = a.tcpHub.bind(state.peer, number(user["user_id"]), cl); err != nil {
				return nil, "", err
			}
		}
		return M{
			"sessionId":        cl.SID,
			"userInfo":         a.safeUser(user),
			"heartbeatSeconds": int64(tcpIdleTimeout.Seconds() / 2),
			"serverTime":       time.Now().UnixMilli(),
		}, "会话绑定成功", nil
	case "config.get":
		if _, _, err := a.tcpAuthenticatedUser(ctx, state); err != nil {
			return nil, "", err
		}
		return a.tcpGlobalConfig(ctx), "配置获取成功", nil
	default:
		return nil, "", clientError{"未知 action", 404}
	}
}

func (a *App) newTCPBusinessRequest(ctx context.Context, remoteAddr, path string, params M) *request {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequestWithContext(ctx, http.MethodPost, path, nil)
	c.Request.RemoteAddr = remoteAddr
	return &request{app: a, c: c, p: params, page: 1}
}

func (a *App) tcpAuthenticatedUser(ctx context.Context, state *tcpConnectionState) (M, claims, error) {
	if state == nil || state.authToken == "" {
		return nil, claims{}, clientError{"请先绑定登录会话", 401}
	}
	user, cl, err := a.authenticate(ctx, state.authToken)
	if err != nil {
		state.authToken = ""
		return nil, claims{}, clientError{"登录状态已失效", 401}
	}
	return user, cl, nil
}

func (a *App) tcpGlobalConfig(ctx context.Context) M {
	system := a.config(ctx, "sysInfo")
	chat := a.config(ctx, "chatInfo")
	compass := a.config(ctx, "compass")
	uploadConfig := a.config(ctx, "fileUpload")
	upload := pick(uploadConfig, "size", "videoSize", "preview", "fileExt")
	upload["directUpload"] = directUploadCapability(uploadConfig)

	system["runMode"] = number(system["runMode"])
	system["showScan"] = number(system["showScan"]) == 1
	revokeSeconds := number(chat["redoTime"])
	if revokeSeconds < 1 {
		revokeSeconds = 120
	}
	chat["revokeMinutes"] = (revokeSeconds + 59) / 60
	chat["maxMessageLength"] = int64(5000)
	compass["status"] = number(compass["status"])
	imageMB := number(upload["size"])
	if imageMB < 1 {
		imageMB = 10
	}
	fileMB := number(upload["videoSize"])
	if fileMB < imageMB {
		fileMB = imageMB
	}
	upload["maxImageBytes"] = imageMB * 1024 * 1024
	upload["maxFileBytes"] = fileMB * 1024 * 1024
	return M{"sysInfo": system, "chatInfo": chat, "compass": compass, "fileUpload": upload}
}

func tcpParams(raw json.RawMessage) (M, error) {
	params := M{}
	if len(raw) == 0 {
		return params, nil
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&params); err != nil {
		return nil, clientError{"请求参数格式错误", 400}
	}
	return params, nil
}

func tcpPublicError(err error) (int, string) {
	var public clientError
	if errors.As(err, &public) {
		code := public.code
		if code <= 0 {
			code = 401
		}
		return code, public.message
	}
	if errors.Is(err, sql.ErrNoRows) {
		return 404, "记录不存在"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return 504, "请求处理超时"
	}
	return 500, "服务器内部错误"
}

func tcpFailure(id string, code int, message string) tcpResponse {
	return tcpResponse{Version: tcpProtocolVersion, ID: id, Code: code, Message: message}
}

func remoteHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	return strings.TrimSpace(remoteAddr)
}
