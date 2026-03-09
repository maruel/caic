// Login screen: shows OAuth provider buttons when auth is enabled.
package com.fghbuild.caic.ui.login

import android.content.Intent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.core.net.toUri

@Composable
fun LoginScreen(
    serverURL: String,
    providers: List<String>,
) {
    val context = LocalContext.current
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text("caic", style = MaterialTheme.typography.displaySmall)
        Spacer(modifier = Modifier.height(8.dp))
        Text("Coding Agents in Containers", style = MaterialTheme.typography.bodyMedium)
        Spacer(modifier = Modifier.height(32.dp))
        providers.forEach { provider ->
            val label = when (provider) {
                "github" -> "Sign in with GitHub"
                "gitlab" -> "Sign in with GitLab"
                else -> "Sign in with $provider"
            }
            Button(
                onClick = {
                    val url = "$serverURL/api/v1/auth/$provider/start?return=app"
                    context.startActivity(Intent(Intent.ACTION_VIEW, url.toUri()))
                },
                modifier = Modifier
                    .widthIn(min = 240.dp)
                    .padding(bottom = 8.dp),
            ) {
                Text(label)
            }
        }
    }
}
