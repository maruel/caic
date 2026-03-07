// Usage badges showing API utilization with color-coded thresholds.
package com.fghbuild.caic.ui.tasklist

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.caic.sdk.v1.ExtraUsage
import com.caic.sdk.v1.UsageResp
import com.caic.sdk.v1.UsageWindow
import com.fghbuild.caic.ui.theme.appColors

private data class BadgeColors(val bg: Color, val fg: Color)

@Composable
private fun windowColors(pct: Int, yellowAt: Int, redAt: Int): BadgeColors {
    val appColors = MaterialTheme.appColors
    val scheme = MaterialTheme.colorScheme
    return when {
        pct >= redAt -> BadgeColors(scheme.errorContainer, scheme.onErrorContainer)
        pct >= yellowAt -> BadgeColors(appColors.warningBg, appColors.warningText)
        else -> BadgeColors(appColors.successBg, appColors.successText)
    }
}

@Composable
fun UsageBadges(usage: UsageResp) {
    Row(horizontalArrangement = Arrangement.spacedBy(4.dp)) {
        WindowBadge(label = "5h", window = usage.fiveHour, yellowAt = 80, redAt = 90)
        WindowBadge(label = "7d", window = usage.sevenDay, yellowAt = 90, redAt = 95)
        ExtraBadge(extra = usage.extraUsage)
    }
}

@Composable
private fun WindowBadge(label: String, window: UsageWindow, yellowAt: Int, redAt: Int) {
    val pct = window.utilization.toInt().coerceIn(0, 100)
    val colors = windowColors(pct, yellowAt, redAt)
    BadgeText(text = "$label: $pct%", colors = colors)
}

@Composable
private fun ExtraBadge(extra: ExtraUsage) {
    val used = extra.usedCredits / 100.0
    val limit = extra.monthlyLimit / 100.0
    if (used == 0.0 && limit == 0.0) return
    val pct = extra.utilization.toInt().coerceIn(0, 100)
    val colors = if (!extra.isEnabled) {
        BadgeColors(MaterialTheme.appColors.badgeDisabledBg, MaterialTheme.colorScheme.secondary)
    } else {
        windowColors(pct, yellowAt = 50, redAt = 80)
    }
    val label = "Extra: $${used.toInt()} / $${limit.toInt()}"
    BadgeText(
        text = label,
        colors = colors,
        strikethrough = !extra.isEnabled,
    )
}

@Composable
private fun BadgeText(
    text: String,
    colors: BadgeColors,
    strikethrough: Boolean = false,
) {
    val style = if (strikethrough) {
        MaterialTheme.typography.labelSmall.merge(
            TextStyle(textDecoration = androidx.compose.ui.text.style.TextDecoration.LineThrough)
        )
    } else {
        MaterialTheme.typography.labelSmall
    }
    Text(
        text = text,
        style = style,
        color = colors.fg,
        fontWeight = FontWeight.Medium,
        modifier = Modifier
            .background(colors.bg, RoundedCornerShape(4.dp))
            .padding(horizontal = 4.dp, vertical = 2.dp),
    )
}
