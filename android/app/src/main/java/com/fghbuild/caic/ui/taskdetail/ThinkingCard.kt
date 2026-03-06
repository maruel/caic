// Collapsed card for an agent thinking block, analogous to ToolCallCard.
package com.fghbuild.caic.ui.taskdetail

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.unit.dp
import com.caic.sdk.v1.EventKinds
import com.caic.sdk.v1.EventMessage

@Composable
fun ThinkingCard(events: List<EventMessage>, modifier: Modifier = Modifier) {
    val text = run {
        val finalEv = events.lastOrNull { it.kind == EventKinds.Thinking }
        if (finalEv?.thinking != null) {
            finalEv.thinking!!.text
        } else {
            events.filter { it.kind == EventKinds.ThinkingDelta && it.thinkingDelta != null }
                .joinToString("") { it.thinkingDelta!!.text }
        }
    }
    if (text.isBlank()) return

    var expanded by rememberSaveable(events.firstOrNull()?.ts) { mutableStateOf(false) }

    Surface(
        modifier = modifier.fillMaxWidth(),
        tonalElevation = 1.dp,
        shape = MaterialTheme.shapes.small,
    ) {
        Column {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { expanded = !expanded }
                    .padding(8.dp),
            ) {
                Text(
                    text = "Thinking",
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    fontStyle = FontStyle.Italic,
                )
            }
            AnimatedVisibility(visible = expanded) {
                Text(
                    text = text,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                )
            }
        }
    }
}
