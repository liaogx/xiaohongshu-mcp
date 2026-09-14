package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func cleanupMCPResult(v any, err error) (*mcp.CallToolResult, any, error) {
	if err != nil {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, nil, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(data)}}}, nil, nil
}

func registerCleanupTools(server *mcp.Server, app *AppServer) {
	mcp.AddTool(server, &mcp.Tool{Name: "prepare_account_cleanup", Description: "盘点八类账号清理范围并保存私有计划，不删除。明确报告平台不可读、历史覆盖不全及手机端才能执行的类别；候选目标执行前再次核验账号和所有权。", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		withPanicRecovery("prepare_account_cleanup", func(ctx context.Context, _ *mcp.CallToolRequest, args CleanupPrepareArgs) (*mcp.CallToolResult, any, error) {
			v, err := app.xiaohongshuService.PrepareAccountCleanup(ctx, args)
			return cleanupMCPResult(v, err)
		}))
	mcp.AddTool(server, &mcp.Tool{Name: "execute_account_cleanup", Description: "执行已明确授权的清理计划，必须传 plan_id 和 confirm:true；每次一个目标，变更至少间隔60秒。支持已核实的笔记删除、评论删除、取消收藏/赞/关注；当前网页不支持群聊或私信清理。无法确认的结果不自动重试。", Annotations: &mcp.ToolAnnotations{DestructiveHint: boolPtr(true)}},
		withPanicRecovery("execute_account_cleanup", func(ctx context.Context, _ *mcp.CallToolRequest, args CleanupExecuteArgs) (*mcp.CallToolResult, any, error) {
			v, err := app.xiaohongshuService.ExecuteAccountCleanup(ctx, args)
			return cleanupMCPResult(v, err)
		}))
	mcp.AddTool(server, &mcp.Tool{Name: "get_account_cleanup_status", Description: "只读获取私有清理计划及实际执行回执；不确定不等于成功，覆盖不全不等于全部清空。", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		withPanicRecovery("get_account_cleanup_status", func(_ context.Context, _ *mcp.CallToolRequest, args CleanupStatusArgs) (*mcp.CallToolResult, any, error) {
			v, err := app.xiaohongshuService.GetAccountCleanupStatus(args.PlanID)
			return cleanupMCPResult(v, err)
		}))
	mcp.AddTool(server, &mcp.Tool{Name: "discover_cleanup_comments", Description: "按已知笔记 ID 查找当前账号已发评论，或自有笔记收到的评论，返回删除候选 ID。只读，不提供账号全部历史；别人在其他笔记的回复不能当作自己拥有的评论删除。", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		withPanicRecovery("discover_cleanup_comments", func(ctx context.Context, _ *mcp.CallToolRequest, args CleanupDiscoverArgs) (*mcp.CallToolResult, any, error) {
			v, err := app.xiaohongshuService.DiscoverCleanupComments(ctx, args)
			return cleanupMCPResult(v, err)
		}))
}

func cleanupHTTPResult(c *gin.Context, v any, err error) {
	if err != nil {
		respondError(c, http.StatusBadRequest, "CLEANUP_FAILED", err.Error(), nil)
		return
	}
	respondSuccess(c, v, "清理请求已处理，完成情况以 data.status 和回执为准")
}
func (s *AppServer) prepareAccountCleanupHandler(c *gin.Context) {
	var args CleanupPrepareArgs
	if err := c.ShouldBindJSON(&args); err != nil {
		respondError(c, 400, "INVALID_REQUEST", "invalid cleanup request", nil)
		return
	}
	v, err := s.xiaohongshuService.PrepareAccountCleanup(c.Request.Context(), args)
	cleanupHTTPResult(c, v, err)
}
func (s *AppServer) executeAccountCleanupHandler(c *gin.Context) {
	var args CleanupExecuteArgs
	if err := c.ShouldBindJSON(&args); err != nil {
		respondError(c, 400, "INVALID_REQUEST", "invalid cleanup request", nil)
		return
	}
	v, err := s.xiaohongshuService.ExecuteAccountCleanup(c.Request.Context(), args)
	cleanupHTTPResult(c, v, err)
}
func (s *AppServer) accountCleanupStatusHandler(c *gin.Context) {
	v, err := s.xiaohongshuService.GetAccountCleanupStatus(c.Query("plan_id"))
	cleanupHTTPResult(c, v, err)
}
func (s *AppServer) discoverCleanupCommentsHandler(c *gin.Context) {
	var args CleanupDiscoverArgs
	if err := c.ShouldBindJSON(&args); err != nil {
		respondError(c, 400, "INVALID_REQUEST", "invalid cleanup request", nil)
		return
	}
	v, err := s.xiaohongshuService.DiscoverCleanupComments(c.Request.Context(), args)
	cleanupHTTPResult(c, v, err)
}
