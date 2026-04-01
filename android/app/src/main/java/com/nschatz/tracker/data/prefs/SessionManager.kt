package com.nschatz.tracker.data.prefs

import android.content.Context
import android.content.SharedPreferences
import com.nschatz.tracker.BuildConfig
import com.nschatz.tracker.data.model.UserInfo

class SessionManager(context: Context) {

    private val prefs: SharedPreferences =
        context.getSharedPreferences("tracker_session", Context.MODE_PRIVATE)

    var token: String?
        get() = prefs.getString(KEY_TOKEN, null)
        set(value) = prefs.edit().putString(KEY_TOKEN, value).apply()

    var userId: String?
        get() = prefs.getString(KEY_USER_ID, null)
        set(value) = prefs.edit().putString(KEY_USER_ID, value).apply()

    var email: String?
        get() = prefs.getString(KEY_EMAIL, null)
        set(value) = prefs.edit().putString(KEY_EMAIL, value).apply()

    var displayName: String?
        get() = prefs.getString(KEY_DISPLAY_NAME, null)
        set(value) = prefs.edit().putString(KEY_DISPLAY_NAME, value).apply()

    var activeCircleId: String?
        get() = prefs.getString(KEY_ACTIVE_CIRCLE_ID, null)
        set(value) = prefs.edit().putString(KEY_ACTIVE_CIRCLE_ID, value).apply()

    var serverUrl: String
        get() = prefs.getString(KEY_SERVER_URL, BuildConfig.API_BASE_URL) ?: BuildConfig.API_BASE_URL
        set(value) = prefs.edit().putString(KEY_SERVER_URL, value).apply()

    val isLoggedIn: Boolean
        get() = token != null && userId != null

    fun saveAuth(token: String, user: UserInfo) {
        prefs.edit()
            .putString(KEY_TOKEN, token)
            .putString(KEY_USER_ID, user.id)
            .putString(KEY_EMAIL, user.email)
            .putString(KEY_DISPLAY_NAME, user.displayName)
            .apply()
    }

    fun clear() {
        prefs.edit()
            .remove(KEY_TOKEN)
            .remove(KEY_USER_ID)
            .remove(KEY_EMAIL)
            .remove(KEY_DISPLAY_NAME)
            .remove(KEY_ACTIVE_CIRCLE_ID)
            .apply()
    }

    companion object {
        private const val KEY_TOKEN = "token"
        private const val KEY_USER_ID = "user_id"
        private const val KEY_EMAIL = "email"
        private const val KEY_DISPLAY_NAME = "display_name"
        private const val KEY_ACTIVE_CIRCLE_ID = "active_circle_id"
        private const val KEY_SERVER_URL = "server_url"
    }
}
