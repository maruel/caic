// Renders a text group: combines textDelta fragments, renders markdown or isolated HTML.
package com.fghbuild.caic.ui.taskdetail

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.caic.sdk.v1.EventMessage
import com.caic.sdk.v1.EventKinds
import com.fghbuild.caic.ui.theme.markdownTypography
import com.fghbuild.caic.util.GroupKind
import com.fghbuild.caic.util.MessageGroup
import com.mikepenz.markdown.m3.Markdown

/** Returns true when text is a raw HTML fragment (weaker model dumped HTML as text). */
private fun looksLikeHTML(text: String): Boolean {
    val trimmed = text.trimStart()
    return trimmed.startsWith("<style") ||
        trimmed.startsWith("<div") ||
        trimmed.startsWith("<!--")
}

@Composable
fun TextMessageGroup(events: List<EventMessage>) {
    val thinkingEvents = remember(events) {
        events.filter { it.kind == EventKinds.Thinking || it.kind == EventKinds.ThinkingDelta }
    }
    val text = remember(events) {
        val finalEv = events.lastOrNull { it.kind == EventKinds.Text }
        if (finalEv?.text != null) {
            finalEv.text!!.text
        } else {
            events
                .filter { it.kind == EventKinds.TextDelta && it.textDelta != null }
                .joinToString("") { it.textDelta!!.text }
        }
    }
    if (thinkingEvents.isEmpty() && text.isBlank()) return
    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
        if (thinkingEvents.isNotEmpty()) {
            ThinkingCard(events = thinkingEvents)
        }
        if (text.isNotBlank()) {
            if (looksLikeHTML(text)) {
                val widgetGroup = remember(text, events) {
                    MessageGroup(
                        kind = GroupKind.WIDGET,
                        events = events,
                        widgetHTML = text,
                        widgetDone = events.any { it.kind == EventKinds.Text },
                    )
                }
                WidgetCard(group = widgetGroup)
            } else {
                Markdown(
                    content = text,
                    modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp),
                    typography = markdownTypography(),
                    colors = com.mikepenz.markdown.m3.markdownColor(
                        text = MaterialTheme.colorScheme.onSurface,
                        codeBackground = MaterialTheme.colorScheme.surfaceVariant,
                    ),
                )
            }
        }
    }
}
