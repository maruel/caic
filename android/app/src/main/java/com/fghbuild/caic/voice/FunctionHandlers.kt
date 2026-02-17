// Dispatches Gemini function calls to the caic API.
package com.fghbuild.caic.voice

import com.caic.sdk.ApiClient
import com.caic.sdk.CreateTaskReq
import com.caic.sdk.InputReq
import com.caic.sdk.RestartReq
import com.caic.sdk.SyncReq
import com.caic.sdk.TaskJSON
import com.fghbuild.caic.util.formatCost
import com.fghbuild.caic.util.formatElapsed
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

class FunctionHandlers(private val apiClient: ApiClient) {

    var onSetActiveTask: ((String) -> Unit)? = null

    suspend fun handle(name: String, args: JsonObject): JsonElement {
        return try {
            when (name) {
                "list_tasks" -> handleListTasks()
                "create_task" -> handleCreateTask(args)
                "get_task_detail" -> handleGetTaskDetail(args)
                "send_message" -> handleSendMessage(args)
                "answer_question" -> handleAnswerQuestion(args)
                "sync_task" -> handleSyncTask(args)
                "terminate_task" -> handleTerminateTask(args)
                "restart_task" -> handleRestartTask(args)
                "get_usage" -> handleGetUsage()
                "set_active_task" -> handleSetActiveTask(args)
                "list_repos" -> handleListRepos()
                else -> errorResult("Unknown function: $name")
            }
        } catch (@Suppress("TooGenericExceptionCaught") e: Exception) {
            errorResult("Error: ${e.message}")
        }
    }

    private suspend fun handleListTasks(): JsonElement {
        val tasks = apiClient.listTasks()
        if (tasks.isEmpty()) {
            return JsonObject(mapOf("summary" to JsonPrimitive("No tasks running.")))
        }
        val lines = tasks.joinToString("\n") { t -> taskSummaryLine(t) }
        return JsonObject(
            mapOf(
                "count" to JsonPrimitive(tasks.size),
                "tasks" to JsonPrimitive(lines),
            )
        )
    }

    private suspend fun handleCreateTask(args: JsonObject): JsonElement {
        val prompt = args.requireString("prompt")
        val repo = args.requireString("repo")
        val model = args.optString("model")
        val harness = args.optString("harness") ?: "claude"
        val resp = apiClient.createTask(
            CreateTaskReq(
                prompt = prompt,
                repo = repo,
                model = model,
                harness = harness,
            )
        )
        return JsonObject(
            mapOf(
                "status" to JsonPrimitive(resp.status),
                "id" to JsonPrimitive(resp.id),
            )
        )
    }

    private suspend fun handleGetTaskDetail(args: JsonObject): JsonElement {
        val taskId = args.requireString("task_id")
        val tasks = apiClient.listTasks()
        val t = tasks.find { it.id == taskId }
            ?: return errorResult("Task not found: $taskId")
        val detail = buildString {
            appendLine("Prompt: ${t.task.trim()}")
            appendLine("State: ${t.state}  Elapsed: ${formatElapsed(t.durationMs)}  Cost: ${formatCost(t.costUSD)}")
            when {
                t.state == "asking" -> appendLine("Waiting for user input before it can continue.")
                t.state == "terminated" && !t.result.isNullOrBlank() ->
                    appendLine("Result: ${t.result}")
                t.state == "failed" ->
                    appendLine("Error: ${t.error ?: "unknown"}")
            }
            t.diffStat?.takeIf { it.isNotEmpty() }?.let { diff ->
                append("Changed: ${diff.joinToString(", ") { it.path }}")
            }
        }.trim()
        return JsonObject(mapOf("detail" to JsonPrimitive(detail)))
    }

    private suspend fun handleSendMessage(args: JsonObject): JsonElement {
        val taskId = args.requireString("task_id")
        val message = args.requireString("message")
        val resp = apiClient.sendInput(taskId, InputReq(prompt = message))
        return JsonObject(mapOf("status" to JsonPrimitive(resp.status)))
    }

    private suspend fun handleAnswerQuestion(args: JsonObject): JsonElement {
        val taskId = args.requireString("task_id")
        val answer = args.requireString("answer")
        val resp = apiClient.sendInput(taskId, InputReq(prompt = answer))
        return JsonObject(mapOf("status" to JsonPrimitive(resp.status)))
    }

    private suspend fun handleSyncTask(args: JsonObject): JsonElement {
        val taskId = args.requireString("task_id")
        val force = args["force"]?.jsonPrimitive?.booleanOrNull ?: false
        val resp = apiClient.syncTask(taskId, SyncReq(force = force))
        val result = mutableMapOf<String, JsonElement>(
            "status" to JsonPrimitive(resp.status),
        )
        resp.safetyIssues?.let { issues ->
            result["safetyIssues"] = JsonArray(
                issues.map { issue ->
                    JsonObject(
                        mapOf(
                            "file" to JsonPrimitive(issue.file),
                            "kind" to JsonPrimitive(issue.kind),
                            "detail" to JsonPrimitive(issue.detail),
                        )
                    )
                }
            )
        }
        return JsonObject(result)
    }

    private suspend fun handleTerminateTask(args: JsonObject): JsonElement {
        val taskId = args.requireString("task_id")
        val resp = apiClient.terminateTask(taskId)
        return JsonObject(mapOf("status" to JsonPrimitive(resp.status)))
    }

    private suspend fun handleRestartTask(args: JsonObject): JsonElement {
        val taskId = args.requireString("task_id")
        val prompt = args.requireString("prompt")
        val resp = apiClient.restartTask(taskId, RestartReq(prompt = prompt))
        return JsonObject(mapOf("status" to JsonPrimitive(resp.status)))
    }

    private suspend fun handleGetUsage(): JsonElement {
        val usage = apiClient.getUsage()
        fun pct(v: Double) = "${(v * 100).toInt()}%"
        val summary = buildString {
            appendLine("5-hour window: ${pct(usage.fiveHour.utilization)} used, resets ${usage.fiveHour.resetsAt}")
            append("7-day window: ${pct(usage.sevenDay.utilization)} used, resets ${usage.sevenDay.resetsAt}")
            if (usage.extraUsage.isEnabled) {
                appendLine()
                append(
                    "Extra usage: ${pct(usage.extraUsage.utilization)} of " +
                        "\$${usage.extraUsage.monthlyLimit.toInt()} monthly limit used",
                )
            }
        }
        return JsonObject(mapOf("usage" to JsonPrimitive(summary)))
    }

    @Suppress("UnusedPrivateMember")
    private fun handleSetActiveTask(args: JsonObject): JsonElement {
        val taskId = args.requireString("task_id")
        onSetActiveTask?.invoke(taskId)
        return JsonObject(mapOf("status" to JsonPrimitive("navigated")))
    }

    private suspend fun handleListRepos(): JsonElement {
        val repos = apiClient.listRepos()
        return JsonObject(
            mapOf(
                "repos" to JsonArray(
                    repos.map { r ->
                        JsonObject(
                            mapOf(
                                "path" to JsonPrimitive(r.path),
                                "baseBranch" to JsonPrimitive(r.baseBranch),
                            )
                        )
                    }
                ),
            )
        )
    }
}

private fun taskSummaryLine(t: TaskJSON): String {
    val name = t.task.lines().firstOrNull()?.take(40) ?: t.id
    val base = "[$name] ${t.state} — ${formatElapsed(t.durationMs)}, " +
        "${formatCost(t.costUSD)}, ${t.harness}, repo: ${t.repo}"
    return when {
        t.state == "asking" -> "$base — NEEDS INPUT"
        t.state == "terminated" && !t.result.isNullOrBlank() ->
            "$base — Result: ${t.result!!.take(120)}"
        t.state == "failed" -> "$base — Error: ${t.error ?: "unknown"}"
        else -> base
    }
}

private fun JsonObject.requireString(key: String): String =
    this[key]?.jsonPrimitive?.content
        ?: throw IllegalArgumentException("Missing required parameter: $key")

private fun JsonObject.optString(key: String): String? =
    this[key]?.jsonPrimitive?.content

private fun errorResult(message: String): JsonElement =
    JsonObject(mapOf("error" to JsonPrimitive(message)))
