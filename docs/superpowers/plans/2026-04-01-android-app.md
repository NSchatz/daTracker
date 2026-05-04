# Android App Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a native Kotlin Android app for real-time location sharing, geofence alerts, and location history — the client for our Go backend.

**Architecture:** Single-activity app with fragment navigation. Retrofit for API, Room for offline queue, osmdroid for maps, OkHttp WebSocket for real-time, foreground service for location tracking with adaptive intervals via Activity Recognition API.

**Tech Stack:** Kotlin, Android SDK 34, minSdk 26, Retrofit 2, OkHttp 4, Room, osmdroid, Google Play Services (Location, Activity Recognition), Firebase Cloud Messaging, ViewBinding

---

## File Structure

```
android/
  app/
    build.gradle.kts
    src/main/
      AndroidManifest.xml
      java/com/nschatz/tracker/
        TrackerApp.kt                          - Application class, DI setup
        data/
          api/
            ApiClient.kt                       - Retrofit + OkHttp setup
            AuthApi.kt                         - POST /auth/register, /auth/login
            LocationApi.kt                     - POST /locations, GET /latest, /history
            CircleApi.kt                       - CRUD /circles
            GeofenceApi.kt                     - CRUD /geofences
            FcmApi.kt                          - POST /fcm-token
            TokenInterceptor.kt                - Injects Bearer token into requests
          model/
            ApiModels.kt                       - Request/response data classes
          local/
            TrackerDatabase.kt                 - Room database
            PendingLocationDao.kt              - DAO for offline queue
            CachedLocationDao.kt               - DAO for cached member positions
            Entities.kt                        - Room entity classes
          prefs/
            SessionManager.kt                  - SharedPreferences for JWT + user info
          repository/
            AuthRepository.kt                  - Login/register, token management
            LocationRepository.kt              - Submit locations, query history, offline queue
            CircleRepository.kt                - Circle CRUD
            GeofenceRepository.kt              - Geofence CRUD
        service/
          LocationService.kt                   - Foreground service, Fused Location Provider
          ActivityTransitionReceiver.kt        - Detects stationary/walking/driving
        websocket/
          LocationWebSocketClient.kt           - OkHttp WebSocket, auto-reconnect
        fcm/
          TrackerFirebaseService.kt            - FCM message handler
        ui/
          auth/
            LoginActivity.kt                   - Login screen (entry point when unauthenticated)
            RegisterActivity.kt                - Registration screen
          main/
            MainActivity.kt                    - Hosts bottom nav + fragments
          map/
            MapFragment.kt                     - Live map with member markers + geofences
          history/
            HistoryFragment.kt                 - Member path on map with date picker
          places/
            PlacesFragment.kt                  - Geofence list + create/edit
          circle/
            CircleFragment.kt                  - Member list, invite sharing
          profile/
            ProfileFragment.kt                 - Display name, notification prefs
      res/
        layout/
          activity_login.xml
          activity_register.xml
          activity_main.xml
          fragment_map.xml
          fragment_history.xml
          fragment_places.xml
          fragment_circle.xml
          fragment_profile.xml
          item_member.xml
          item_geofence.xml
          dialog_geofence_edit.xml
        menu/
          bottom_nav.xml
        values/
          strings.xml
          themes.xml
          colors.xml
        drawable/
          ic_map.xml
          ic_history.xml
          ic_place.xml
          ic_group.xml
          ic_person.xml
        xml/
          network_security_config.xml
    src/test/
      java/com/nschatz/tracker/
        data/repository/
          AuthRepositoryTest.kt
          LocationRepositoryTest.kt
  build.gradle.kts                             - Project-level Gradle
  settings.gradle.kts
  gradle.properties
```

---

## Task 1: Project Scaffolding

**Files:**
- Create: `android/build.gradle.kts`
- Create: `android/settings.gradle.kts`
- Create: `android/gradle.properties`
- Create: `android/app/build.gradle.kts`
- Create: `android/app/src/main/AndroidManifest.xml`
- Create: `android/app/src/main/java/com/nschatz/tracker/TrackerApp.kt`
- Create: `android/app/src/main/res/values/strings.xml`
- Create: `android/app/src/main/res/values/themes.xml`
- Create: `android/app/src/main/res/values/colors.xml`
- Create: `android/app/src/main/res/xml/network_security_config.xml`

- [ ] **Step 1: Create project-level build.gradle.kts**

```kotlin
// android/build.gradle.kts
plugins {
    id("com.android.application") version "8.5.0" apply false
    id("org.jetbrains.kotlin.android") version "2.0.0" apply false
    id("com.google.gms.google-services") version "4.4.2" apply false
}
```

- [ ] **Step 2: Create settings.gradle.kts**

```kotlin
// android/settings.gradle.kts
pluginManagement {
    repositories {
        google()
        mavenCentral()
        gradlePluginPortal()
    }
}
dependencyResolution {
    repositories {
        google()
        mavenCentral()
    }
}

rootProject.name = "Tracker"
include(":app")
```

- [ ] **Step 3: Create gradle.properties**

```properties
# android/gradle.properties
android.useAndroidX=true
org.gradle.jvmargs=-Xmx2048m
android.nonTransitiveRClass=true
```

- [ ] **Step 4: Create app/build.gradle.kts**

```kotlin
// android/app/build.gradle.kts
plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("com.google.gms.google-services")
    id("kotlin-kapt")
}

android {
    namespace = "com.nschatz.tracker"
    compileSdk = 34

    defaultConfig {
        applicationId = "com.nschatz.tracker"
        minSdk = 26
        targetSdk = 34
        versionCode = 1
        versionName = "1.0"

        buildConfigField("String", "API_BASE_URL", "\"http://10.0.2.2:8080\"")
    }

    buildFeatures {
        viewBinding = true
        buildConfig = true
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }
}

dependencies {
    // AndroidX
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.appcompat:appcompat:1.7.0")
    implementation("com.google.android.material:material:1.12.0")
    implementation("androidx.constraintlayout:constraintlayout:2.1.4")
    implementation("androidx.fragment:fragment-ktx:1.8.1")
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.8.3")
    implementation("androidx.lifecycle:lifecycle-viewmodel-ktx:2.8.3")

    // Retrofit + OkHttp
    implementation("com.squareup.retrofit2:retrofit:2.11.0")
    implementation("com.squareup.retrofit2:converter-gson:2.11.0")
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("com.squareup.okhttp3:logging-interceptor:4.12.0")

    // Room
    implementation("androidx.room:room-runtime:2.6.1")
    implementation("androidx.room:room-ktx:2.6.1")
    kapt("androidx.room:room-compiler:2.6.1")

    // osmdroid
    implementation("org.osmdroid:osmdroid-android:6.1.18")

    // Google Play Services
    implementation("com.google.android.gms:play-services-location:21.3.0")

    // Firebase
    implementation(platform("com.google.firebase:firebase-bom:33.1.2"))
    implementation("com.google.firebase:firebase-messaging")

    // Coroutines
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.8.1")

    // Gson
    implementation("com.google.code.gson:gson:2.11.0")

    // Testing
    testImplementation("junit:junit:4.13.2")
    testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-test:1.8.1")
    testImplementation("org.mockito.kotlin:mockito-kotlin:5.4.0")
}
```

- [ ] **Step 5: Create AndroidManifest.xml**

```xml
<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android">

    <uses-permission android:name="android.permission.INTERNET" />
    <uses-permission android:name="android.permission.ACCESS_FINE_LOCATION" />
    <uses-permission android:name="android.permission.ACCESS_COARSE_LOCATION" />
    <uses-permission android:name="android.permission.ACCESS_BACKGROUND_LOCATION" />
    <uses-permission android:name="android.permission.FOREGROUND_SERVICE" />
    <uses-permission android:name="android.permission.FOREGROUND_SERVICE_LOCATION" />
    <uses-permission android:name="android.permission.POST_NOTIFICATIONS" />
    <uses-permission android:name="android.permission.ACTIVITY_RECOGNITION" />
    <uses-permission android:name="android.permission.ACCESS_NETWORK_STATE" />
    <uses-permission android:name="android.permission.WRITE_EXTERNAL_STORAGE"
        android:maxSdkVersion="28" />

    <application
        android:name=".TrackerApp"
        android:allowBackup="true"
        android:icon="@mipmap/ic_launcher"
        android:label="@string/app_name"
        android:supportsRtl="true"
        android:theme="@style/Theme.Tracker"
        android:networkSecurityConfig="@xml/network_security_config">

        <activity
            android:name=".ui.auth.LoginActivity"
            android:exported="true">
            <intent-filter>
                <action android:name="android.intent.action.MAIN" />
                <category android:name="android.intent.category.LAUNCHER" />
            </intent-filter>
        </activity>

        <activity android:name=".ui.auth.RegisterActivity" />
        <activity android:name=".ui.main.MainActivity" />

        <service
            android:name=".service.LocationService"
            android:foregroundServiceType="location" />

        <receiver
            android:name=".service.ActivityTransitionReceiver"
            android:exported="false" />

        <service
            android:name=".fcm.TrackerFirebaseService"
            android:exported="false">
            <intent-filter>
                <action android:name="com.google.firebase.MESSAGING_EVENT" />
            </intent-filter>
        </service>
    </application>
</manifest>
```

- [ ] **Step 6: Create TrackerApp.kt**

```kotlin
package com.nschatz.tracker

import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import org.osmdroid.config.Configuration

class TrackerApp : Application() {

    companion object {
        const val CHANNEL_LOCATION = "location_service"
        const val CHANNEL_PLACE_ALERTS = "place_alerts"
    }

    override fun onCreate() {
        super.onCreate()

        Configuration.getInstance().userAgentValue = packageName

        createNotificationChannels()
    }

    private fun createNotificationChannels() {
        val manager = getSystemService(NotificationManager::class.java)

        val locationChannel = NotificationChannel(
            CHANNEL_LOCATION, "Location Service",
            NotificationManager.IMPORTANCE_LOW
        ).apply { description = "Shows when location tracking is active" }

        val alertChannel = NotificationChannel(
            CHANNEL_PLACE_ALERTS, "Place Alerts",
            NotificationManager.IMPORTANCE_HIGH
        ).apply { description = "Geofence enter/leave notifications" }

        manager.createNotificationChannels(listOf(locationChannel, alertChannel))
    }
}
```

- [ ] **Step 7: Create resource files**

`android/app/src/main/res/values/strings.xml`:
```xml
<resources>
    <string name="app_name">Tracker</string>
    <string name="nav_map">Map</string>
    <string name="nav_history">History</string>
    <string name="nav_places">Places</string>
    <string name="nav_circle">Circle</string>
    <string name="nav_profile">Profile</string>
</resources>
```

`android/app/src/main/res/values/colors.xml`:
```xml
<?xml version="1.0" encoding="utf-8"?>
<resources>
    <color name="primary">#1B73E8</color>
    <color name="primary_dark">#0D57B8</color>
    <color name="accent">#4CAF50</color>
    <color name="background">#FAFAFA</color>
    <color name="surface">#FFFFFF</color>
    <color name="on_primary">#FFFFFF</color>
    <color name="on_surface">#212121</color>
    <color name="geofence_fill">#331B73E8</color>
    <color name="geofence_stroke">#881B73E8</color>
</resources>
```

`android/app/src/main/res/values/themes.xml`:
```xml
<?xml version="1.0" encoding="utf-8"?>
<resources>
    <style name="Theme.Tracker" parent="Theme.Material3.Light.NoActionBar">
        <item name="colorPrimary">@color/primary</item>
        <item name="colorPrimaryDark">@color/primary_dark</item>
        <item name="colorAccent">@color/accent</item>
    </style>
</resources>
```

`android/app/src/main/res/xml/network_security_config.xml`:
```xml
<?xml version="1.0" encoding="utf-8"?>
<network-security-config>
    <domain-config cleartextTrafficPermitted="true">
        <domain includeSubdomains="true">10.0.2.2</domain>
        <domain includeSubdomains="true">localhost</domain>
    </domain-config>
</network-security-config>
```

- [ ] **Step 8: Verify project builds**

```bash
cd android && ./gradlew assembleDebug
```

- [ ] **Step 9: Commit**

```bash
git add android/
git commit -m "feat: Android project scaffolding with dependencies and manifest"
```

---

## Task 2: API Models + Retrofit Client

**Files:**
- Create: `android/app/src/main/java/com/nschatz/tracker/data/model/ApiModels.kt`
- Create: `android/app/src/main/java/com/nschatz/tracker/data/prefs/SessionManager.kt`
- Create: `android/app/src/main/java/com/nschatz/tracker/data/api/TokenInterceptor.kt`
- Create: `android/app/src/main/java/com/nschatz/tracker/data/api/ApiClient.kt`
- Create: `android/app/src/main/java/com/nschatz/tracker/data/api/AuthApi.kt`
- Create: `android/app/src/main/java/com/nschatz/tracker/data/api/LocationApi.kt`
- Create: `android/app/src/main/java/com/nschatz/tracker/data/api/CircleApi.kt`
- Create: `android/app/src/main/java/com/nschatz/tracker/data/api/GeofenceApi.kt`
- Create: `android/app/src/main/java/com/nschatz/tracker/data/api/FcmApi.kt`

- [ ] **Step 1: Create API data models**

```kotlin
// data/model/ApiModels.kt
package com.nschatz.tracker.data.model

import com.google.gson.annotations.SerializedName

// Auth
data class RegisterRequest(
    val email: String,
    @SerializedName("display_name") val displayName: String,
    val password: String,
    @SerializedName("invite_code") val inviteCode: String
)

data class LoginRequest(val email: String, val password: String)

data class AuthResponse(
    val token: String,
    val user: UserInfo
)

data class UserInfo(
    val id: String,
    val email: String,
    @SerializedName("display_name") val displayName: String
)

// Locations
data class LocationBatch(val locations: List<LocationPoint>)

data class LocationPoint(
    val lat: Double,
    val lng: Double,
    val speed: Float? = null,
    @SerializedName("battery_level") val batteryLevel: Int? = null,
    val accuracy: Float? = null,
    @SerializedName("recorded_at") val recordedAt: String
)

data class MemberLocation(
    val id: Long = 0,
    @SerializedName("user_id") val userId: String,
    val lat: Double,
    val lng: Double,
    val speed: Float? = null,
    @SerializedName("battery_level") val batteryLevel: Int? = null,
    val accuracy: Float? = null,
    @SerializedName("recorded_at") val recordedAt: String
)

// Circles
data class CreateCircleRequest(val name: String)

data class JoinCircleRequest(@SerializedName("invite_code") val inviteCode: String)

data class Circle(
    val id: String,
    val name: String,
    @SerializedName("invite_code") val inviteCode: String,
    @SerializedName("created_by") val createdBy: String,
    @SerializedName("created_at") val createdAt: String
)

data class CircleMember(
    @SerializedName("circle_id") val circleId: String,
    @SerializedName("user_id") val userId: String,
    val role: String,
    @SerializedName("joined_at") val joinedAt: String,
    @SerializedName("display_name") val displayName: String = "",
    val email: String = ""
)

// Geofences
data class CreateGeofenceRequest(
    @SerializedName("circle_id") val circleId: String,
    val name: String,
    val lat: Double,
    val lng: Double,
    @SerializedName("radius_meters") val radiusMeters: Float
)

data class UpdateGeofenceRequest(
    val name: String,
    val lat: Double,
    val lng: Double,
    @SerializedName("radius_meters") val radiusMeters: Float
)

data class Geofence(
    val id: String,
    @SerializedName("circle_id") val circleId: String,
    val name: String,
    val lat: Double,
    val lng: Double,
    @SerializedName("radius_meters") val radiusMeters: Float,
    @SerializedName("created_by") val createdBy: String,
    @SerializedName("created_at") val createdAt: String
)

// FCM
data class FcmTokenRequest(val token: String)
```

- [ ] **Step 2: Create SessionManager**

```kotlin
// data/prefs/SessionManager.kt
package com.nschatz.tracker.data.prefs

import android.content.Context
import android.content.SharedPreferences
import com.nschatz.tracker.data.model.UserInfo

class SessionManager(context: Context) {

    private val prefs: SharedPreferences =
        context.getSharedPreferences("tracker_session", Context.MODE_PRIVATE)

    var token: String?
        get() = prefs.getString("token", null)
        set(value) = prefs.edit().putString("token", value).apply()

    var userId: String?
        get() = prefs.getString("user_id", null)
        set(value) = prefs.edit().putString("user_id", value).apply()

    var email: String?
        get() = prefs.getString("email", null)
        set(value) = prefs.edit().putString("email", value).apply()

    var displayName: String?
        get() = prefs.getString("display_name", null)
        set(value) = prefs.edit().putString("display_name", value).apply()

    var activeCircleId: String?
        get() = prefs.getString("active_circle_id", null)
        set(value) = prefs.edit().putString("active_circle_id", value).apply()

    var serverUrl: String
        get() = prefs.getString("server_url", null) ?: com.nschatz.tracker.BuildConfig.API_BASE_URL
        set(value) = prefs.edit().putString("server_url", value).apply()

    val isLoggedIn: Boolean get() = token != null

    fun saveAuth(token: String, user: UserInfo) {
        this.token = token
        this.userId = user.id
        this.email = user.email
        this.displayName = user.displayName
    }

    fun clear() {
        prefs.edit().clear().apply()
    }
}
```

- [ ] **Step 3: Create TokenInterceptor**

```kotlin
// data/api/TokenInterceptor.kt
package com.nschatz.tracker.data.api

import com.nschatz.tracker.data.prefs.SessionManager
import okhttp3.Interceptor
import okhttp3.Response

class TokenInterceptor(private val session: SessionManager) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val request = chain.request()
        val token = session.token

        return if (token != null) {
            val authenticatedRequest = request.newBuilder()
                .header("Authorization", "Bearer $token")
                .build()
            chain.proceed(authenticatedRequest)
        } else {
            chain.proceed(request)
        }
    }
}
```

- [ ] **Step 4: Create ApiClient**

```kotlin
// data/api/ApiClient.kt
package com.nschatz.tracker.data.api

import com.nschatz.tracker.data.prefs.SessionManager
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import java.util.concurrent.TimeUnit

class ApiClient(private val session: SessionManager) {

    private val okHttpClient: OkHttpClient by lazy {
        OkHttpClient.Builder()
            .addInterceptor(TokenInterceptor(session))
            .addInterceptor(HttpLoggingInterceptor().apply {
                level = HttpLoggingInterceptor.Level.BODY
            })
            .connectTimeout(15, TimeUnit.SECONDS)
            .readTimeout(15, TimeUnit.SECONDS)
            .build()
    }

    private val retrofit: Retrofit by lazy {
        Retrofit.Builder()
            .baseUrl(session.serverUrl.trimEnd('/') + "/")
            .client(okHttpClient)
            .addConverterFactory(GsonConverterFactory.create())
            .build()
    }

    val auth: AuthApi by lazy { retrofit.create(AuthApi::class.java) }
    val locations: LocationApi by lazy { retrofit.create(LocationApi::class.java) }
    val circles: CircleApi by lazy { retrofit.create(CircleApi::class.java) }
    val geofences: GeofenceApi by lazy { retrofit.create(GeofenceApi::class.java) }
    val fcm: FcmApi by lazy { retrofit.create(FcmApi::class.java) }

    val rawOkHttpClient: OkHttpClient get() = okHttpClient
}
```

- [ ] **Step 5: Create API interfaces**

`data/api/AuthApi.kt`:
```kotlin
package com.nschatz.tracker.data.api

import com.nschatz.tracker.data.model.*
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.POST

interface AuthApi {
    @POST("auth/register")
    suspend fun register(@Body request: RegisterRequest): Response<AuthResponse>

    @POST("auth/login")
    suspend fun login(@Body request: LoginRequest): Response<AuthResponse>
}
```

`data/api/LocationApi.kt`:
```kotlin
package com.nschatz.tracker.data.api

import com.nschatz.tracker.data.model.*
import retrofit2.Response
import retrofit2.http.*

interface LocationApi {
    @POST("locations")
    suspend fun postLocations(@Body batch: LocationBatch): Response<Unit>

    @GET("locations/latest")
    suspend fun getLatest(@Query("circle_id") circleId: String): Response<List<MemberLocation>>

    @GET("locations/history")
    suspend fun getHistory(
        @Query("user_id") userId: String,
        @Query("from") from: String,
        @Query("to") to: String
    ): Response<List<MemberLocation>>
}
```

`data/api/CircleApi.kt`:
```kotlin
package com.nschatz.tracker.data.api

import com.nschatz.tracker.data.model.*
import retrofit2.Response
import retrofit2.http.*

interface CircleApi {
    @POST("circles")
    suspend fun create(@Body request: CreateCircleRequest): Response<Circle>

    @GET("circles")
    suspend fun getAll(): Response<List<Circle>>

    @POST("circles/{id}/join")
    suspend fun join(@Path("id") id: String, @Body request: JoinCircleRequest): Response<Unit>

    @GET("circles/{id}/members")
    suspend fun getMembers(@Path("id") id: String): Response<List<CircleMember>>
}
```

`data/api/GeofenceApi.kt`:
```kotlin
package com.nschatz.tracker.data.api

import com.nschatz.tracker.data.model.*
import retrofit2.Response
import retrofit2.http.*

interface GeofenceApi {
    @POST("geofences")
    suspend fun create(@Body request: CreateGeofenceRequest): Response<Geofence>

    @GET("geofences")
    suspend fun getAll(@Query("circle_id") circleId: String): Response<List<Geofence>>

    @PUT("geofences/{id}")
    suspend fun update(@Path("id") id: String, @Body request: UpdateGeofenceRequest): Response<Geofence>

    @DELETE("geofences/{id}")
    suspend fun delete(@Path("id") id: String): Response<Unit>
}
```

`data/api/FcmApi.kt`:
```kotlin
package com.nschatz.tracker.data.api

import com.nschatz.tracker.data.model.FcmTokenRequest
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.POST

interface FcmApi {
    @POST("fcm-token")
    suspend fun registerToken(@Body request: FcmTokenRequest): Response<Unit>
}
```

- [ ] **Step 6: Verify build**

```bash
cd android && ./gradlew compileDebugKotlin
```

- [ ] **Step 7: Commit**

```bash
git add android/
git commit -m "feat: API models, Retrofit client, session manager, and token interceptor"
```

---

## Task 3: Room Database + Offline Queue

**Files:**
- Create: `android/app/src/main/java/com/nschatz/tracker/data/local/Entities.kt`
- Create: `android/app/src/main/java/com/nschatz/tracker/data/local/PendingLocationDao.kt`
- Create: `android/app/src/main/java/com/nschatz/tracker/data/local/CachedLocationDao.kt`
- Create: `android/app/src/main/java/com/nschatz/tracker/data/local/TrackerDatabase.kt`

- [ ] **Step 1: Create Room entities**

```kotlin
// data/local/Entities.kt
package com.nschatz.tracker.data.local

import androidx.room.Entity
import androidx.room.PrimaryKey

@Entity(tableName = "pending_locations")
data class PendingLocationEntity(
    @PrimaryKey(autoGenerate = true) val id: Long = 0,
    val lat: Double,
    val lng: Double,
    val speed: Float?,
    val batteryLevel: Int?,
    val accuracy: Float?,
    val recordedAt: String // RFC3339
)

@Entity(tableName = "cached_locations")
data class CachedLocationEntity(
    @PrimaryKey val oderId: String, // synthetic: "{userId}-latest"
    val userId: String,
    val displayName: String,
    val lat: Double,
    val lng: Double,
    val batteryLevel: Int?,
    val recordedAt: String,
    val updatedAt: Long = System.currentTimeMillis()
)
```

- [ ] **Step 2: Create DAOs**

`data/local/PendingLocationDao.kt`:
```kotlin
package com.nschatz.tracker.data.local

import androidx.room.*

@Dao
interface PendingLocationDao {
    @Insert
    suspend fun insert(location: PendingLocationEntity)

    @Insert
    suspend fun insertAll(locations: List<PendingLocationEntity>)

    @Query("SELECT * FROM pending_locations ORDER BY id ASC LIMIT :limit")
    suspend fun getOldest(limit: Int): List<PendingLocationEntity>

    @Query("DELETE FROM pending_locations WHERE id IN (:ids)")
    suspend fun deleteByIds(ids: List<Long>)

    @Query("SELECT COUNT(*) FROM pending_locations")
    suspend fun count(): Int
}
```

`data/local/CachedLocationDao.kt`:
```kotlin
package com.nschatz.tracker.data.local

import androidx.room.*

@Dao
interface CachedLocationDao {
    @Upsert
    suspend fun upsert(location: CachedLocationEntity)

    @Upsert
    suspend fun upsertAll(locations: List<CachedLocationEntity>)

    @Query("SELECT * FROM cached_locations ORDER BY updatedAt DESC")
    suspend fun getAll(): List<CachedLocationEntity>

    @Query("DELETE FROM cached_locations")
    suspend fun clear()
}
```

- [ ] **Step 3: Create Room database**

```kotlin
// data/local/TrackerDatabase.kt
package com.nschatz.tracker.data.local

import android.content.Context
import androidx.room.Database
import androidx.room.Room
import androidx.room.RoomDatabase

@Database(
    entities = [PendingLocationEntity::class, CachedLocationEntity::class],
    version = 1,
    exportSchema = false
)
abstract class TrackerDatabase : RoomDatabase() {
    abstract fun pendingLocationDao(): PendingLocationDao
    abstract fun cachedLocationDao(): CachedLocationDao

    companion object {
        @Volatile
        private var INSTANCE: TrackerDatabase? = null

        fun getInstance(context: Context): TrackerDatabase {
            return INSTANCE ?: synchronized(this) {
                INSTANCE ?: Room.databaseBuilder(
                    context.applicationContext,
                    TrackerDatabase::class.java,
                    "tracker.db"
                ).build().also { INSTANCE = it }
            }
        }
    }
}
```

- [ ] **Step 4: Verify build**

```bash
cd android && ./gradlew compileDebugKotlin
```

- [ ] **Step 5: Commit**

```bash
git add android/
git commit -m "feat: Room database with pending location queue and cached positions"
```

---

## Task 4: Repositories

**Files:**
- Create: `android/app/src/main/java/com/nschatz/tracker/data/repository/AuthRepository.kt`
- Create: `android/app/src/main/java/com/nschatz/tracker/data/repository/LocationRepository.kt`
- Create: `android/app/src/main/java/com/nschatz/tracker/data/repository/CircleRepository.kt`
- Create: `android/app/src/main/java/com/nschatz/tracker/data/repository/GeofenceRepository.kt`

- [ ] **Step 1: Create AuthRepository**

```kotlin
// data/repository/AuthRepository.kt
package com.nschatz.tracker.data.repository

import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.model.LoginRequest
import com.nschatz.tracker.data.model.RegisterRequest
import com.nschatz.tracker.data.prefs.SessionManager

class AuthRepository(
    private val api: ApiClient,
    private val session: SessionManager
) {
    suspend fun register(email: String, displayName: String, password: String, inviteCode: String): Result<Unit> {
        return try {
            val response = api.auth.register(RegisterRequest(email, displayName, password, inviteCode))
            if (response.isSuccessful && response.body() != null) {
                val body = response.body()!!
                session.saveAuth(body.token, body.user)
                Result.success(Unit)
            } else {
                Result.failure(Exception(response.errorBody()?.string() ?: "Registration failed"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun login(email: String, password: String): Result<Unit> {
        return try {
            val response = api.auth.login(LoginRequest(email, password))
            if (response.isSuccessful && response.body() != null) {
                val body = response.body()!!
                session.saveAuth(body.token, body.user)
                Result.success(Unit)
            } else {
                Result.failure(Exception(response.errorBody()?.string() ?: "Login failed"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    fun logout() {
        session.clear()
    }
}
```

- [ ] **Step 2: Create LocationRepository**

```kotlin
// data/repository/LocationRepository.kt
package com.nschatz.tracker.data.repository

import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.local.PendingLocationDao
import com.nschatz.tracker.data.local.PendingLocationEntity
import com.nschatz.tracker.data.model.LocationBatch
import com.nschatz.tracker.data.model.LocationPoint
import com.nschatz.tracker.data.model.MemberLocation
import java.time.Instant
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter

class LocationRepository(
    private val api: ApiClient,
    private val pendingDao: PendingLocationDao
) {
    private val formatter = DateTimeFormatter.ISO_INSTANT

    suspend fun submitLocations(points: List<LocationPoint>): Result<Unit> {
        return try {
            val response = api.locations.postLocations(LocationBatch(points))
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Submit failed: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun queueLocation(lat: Double, lng: Double, speed: Float?, battery: Int?, accuracy: Float?) {
        val entity = PendingLocationEntity(
            lat = lat, lng = lng, speed = speed,
            batteryLevel = battery, accuracy = accuracy,
            recordedAt = Instant.now().toString()
        )
        pendingDao.insert(entity)
    }

    suspend fun flushQueue(): Result<Int> {
        val batch = pendingDao.getOldest(50)
        if (batch.isEmpty()) return Result.success(0)

        val points = batch.map {
            LocationPoint(it.lat, it.lng, it.speed, it.batteryLevel, it.accuracy, it.recordedAt)
        }

        return try {
            val response = api.locations.postLocations(LocationBatch(points))
            if (response.isSuccessful) {
                pendingDao.deleteByIds(batch.map { it.id })
                Result.success(batch.size)
            } else {
                Result.failure(Exception("Flush failed: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getLatest(circleId: String): Result<List<MemberLocation>> {
        return try {
            val response = api.locations.getLatest(circleId)
            if (response.isSuccessful) {
                Result.success(response.body() ?: emptyList())
            } else {
                Result.failure(Exception("Get latest failed: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getHistory(userId: String, from: Instant, to: Instant): Result<List<MemberLocation>> {
        return try {
            val response = api.locations.getHistory(
                userId, from.toString(), to.toString()
            )
            if (response.isSuccessful) {
                Result.success(response.body() ?: emptyList())
            } else {
                Result.failure(Exception("Get history failed: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun pendingCount(): Int = pendingDao.count()
}
```

- [ ] **Step 3: Create CircleRepository**

```kotlin
// data/repository/CircleRepository.kt
package com.nschatz.tracker.data.repository

import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.model.*

class CircleRepository(private val api: ApiClient) {

    suspend fun create(name: String): Result<Circle> {
        return try {
            val response = api.circles.create(CreateCircleRequest(name))
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Create circle failed"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getAll(): Result<List<Circle>> {
        return try {
            val response = api.circles.getAll()
            if (response.isSuccessful) {
                Result.success(response.body() ?: emptyList())
            } else {
                Result.failure(Exception("Get circles failed"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun join(circleId: String, inviteCode: String): Result<Unit> {
        return try {
            val response = api.circles.join(circleId, JoinCircleRequest(inviteCode))
            if (response.isSuccessful) Result.success(Unit)
            else Result.failure(Exception("Join failed"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getMembers(circleId: String): Result<List<CircleMember>> {
        return try {
            val response = api.circles.getMembers(circleId)
            if (response.isSuccessful) {
                Result.success(response.body() ?: emptyList())
            } else {
                Result.failure(Exception("Get members failed"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
```

- [ ] **Step 4: Create GeofenceRepository**

```kotlin
// data/repository/GeofenceRepository.kt
package com.nschatz.tracker.data.repository

import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.model.*

class GeofenceRepository(private val api: ApiClient) {

    suspend fun create(circleId: String, name: String, lat: Double, lng: Double, radius: Float): Result<Geofence> {
        return try {
            val response = api.geofences.create(CreateGeofenceRequest(circleId, name, lat, lng, radius))
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Create geofence failed"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getAll(circleId: String): Result<List<Geofence>> {
        return try {
            val response = api.geofences.getAll(circleId)
            if (response.isSuccessful) {
                Result.success(response.body() ?: emptyList())
            } else {
                Result.failure(Exception("Get geofences failed"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun update(id: String, name: String, lat: Double, lng: Double, radius: Float): Result<Geofence> {
        return try {
            val response = api.geofences.update(id, UpdateGeofenceRequest(name, lat, lng, radius))
            if (response.isSuccessful && response.body() != null) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Update geofence failed"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun delete(id: String): Result<Unit> {
        return try {
            val response = api.geofences.delete(id)
            if (response.isSuccessful) Result.success(Unit)
            else Result.failure(Exception("Delete geofence failed"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
```

- [ ] **Step 5: Verify build**

```bash
cd android && ./gradlew compileDebugKotlin
```

- [ ] **Step 6: Commit**

```bash
git add android/
git commit -m "feat: repositories for auth, locations, circles, and geofences"
```

---

## Task 5: Auth Screens (Login + Register)

**Files:**
- Create: `android/app/src/main/res/layout/activity_login.xml`
- Create: `android/app/src/main/res/layout/activity_register.xml`
- Create: `android/app/src/main/java/com/nschatz/tracker/ui/auth/LoginActivity.kt`
- Create: `android/app/src/main/java/com/nschatz/tracker/ui/auth/RegisterActivity.kt`

- [ ] **Step 1: Create login layout**

```xml
<!-- res/layout/activity_login.xml -->
<?xml version="1.0" encoding="utf-8"?>
<LinearLayout xmlns:android="http://schemas.android.com/apk/res/android"
    android:layout_width="match_parent"
    android:layout_height="match_parent"
    android:gravity="center"
    android:orientation="vertical"
    android:padding="32dp">

    <TextView
        android:layout_width="wrap_content"
        android:layout_height="wrap_content"
        android:text="Tracker"
        android:textSize="32sp"
        android:textStyle="bold"
        android:layout_marginBottom="8dp" />

    <com.google.android.material.textfield.TextInputLayout
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:layout_marginBottom="8dp"
        android:hint="Server URL">
        <com.google.android.material.textfield.TextInputEditText
            android:id="@+id/editServerUrl"
            android:layout_width="match_parent"
            android:layout_height="wrap_content"
            android:inputType="textUri" />
    </com.google.android.material.textfield.TextInputLayout>

    <com.google.android.material.textfield.TextInputLayout
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:layout_marginBottom="8dp"
        android:hint="Email">
        <com.google.android.material.textfield.TextInputEditText
            android:id="@+id/editEmail"
            android:layout_width="match_parent"
            android:layout_height="wrap_content"
            android:inputType="textEmailAddress" />
    </com.google.android.material.textfield.TextInputLayout>

    <com.google.android.material.textfield.TextInputLayout
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:layout_marginBottom="16dp"
        android:hint="Password">
        <com.google.android.material.textfield.TextInputEditText
            android:id="@+id/editPassword"
            android:layout_width="match_parent"
            android:layout_height="wrap_content"
            android:inputType="textPassword" />
    </com.google.android.material.textfield.TextInputLayout>

    <com.google.android.material.button.MaterialButton
        android:id="@+id/btnLogin"
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:text="Login" />

    <com.google.android.material.button.MaterialButton
        android:id="@+id/btnRegister"
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:text="Create Account"
        style="@style/Widget.Material3.Button.TextButton" />

    <ProgressBar
        android:id="@+id/progress"
        android:layout_width="wrap_content"
        android:layout_height="wrap_content"
        android:layout_marginTop="16dp"
        android:visibility="gone" />

    <TextView
        android:id="@+id/txtError"
        android:layout_width="wrap_content"
        android:layout_height="wrap_content"
        android:textColor="#D32F2F"
        android:layout_marginTop="8dp"
        android:visibility="gone" />
</LinearLayout>
```

- [ ] **Step 2: Create register layout**

```xml
<!-- res/layout/activity_register.xml -->
<?xml version="1.0" encoding="utf-8"?>
<LinearLayout xmlns:android="http://schemas.android.com/apk/res/android"
    android:layout_width="match_parent"
    android:layout_height="match_parent"
    android:gravity="center"
    android:orientation="vertical"
    android:padding="32dp">

    <TextView
        android:layout_width="wrap_content"
        android:layout_height="wrap_content"
        android:text="Create Account"
        android:textSize="24sp"
        android:textStyle="bold"
        android:layout_marginBottom="16dp" />

    <com.google.android.material.textfield.TextInputLayout
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:layout_marginBottom="8dp"
        android:hint="Display Name">
        <com.google.android.material.textfield.TextInputEditText
            android:id="@+id/editDisplayName"
            android:layout_width="match_parent"
            android:layout_height="wrap_content" />
    </com.google.android.material.textfield.TextInputLayout>

    <com.google.android.material.textfield.TextInputLayout
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:layout_marginBottom="8dp"
        android:hint="Email">
        <com.google.android.material.textfield.TextInputEditText
            android:id="@+id/editEmail"
            android:layout_width="match_parent"
            android:layout_height="wrap_content"
            android:inputType="textEmailAddress" />
    </com.google.android.material.textfield.TextInputLayout>

    <com.google.android.material.textfield.TextInputLayout
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:layout_marginBottom="8dp"
        android:hint="Password">
        <com.google.android.material.textfield.TextInputEditText
            android:id="@+id/editPassword"
            android:layout_width="match_parent"
            android:layout_height="wrap_content"
            android:inputType="textPassword" />
    </com.google.android.material.textfield.TextInputLayout>

    <com.google.android.material.textfield.TextInputLayout
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:layout_marginBottom="16dp"
        android:hint="Invite Code">
        <com.google.android.material.textfield.TextInputEditText
            android:id="@+id/editInviteCode"
            android:layout_width="match_parent"
            android:layout_height="wrap_content" />
    </com.google.android.material.textfield.TextInputLayout>

    <com.google.android.material.button.MaterialButton
        android:id="@+id/btnRegister"
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:text="Register" />

    <ProgressBar
        android:id="@+id/progress"
        android:layout_width="wrap_content"
        android:layout_height="wrap_content"
        android:layout_marginTop="16dp"
        android:visibility="gone" />

    <TextView
        android:id="@+id/txtError"
        android:layout_width="wrap_content"
        android:layout_height="wrap_content"
        android:textColor="#D32F2F"
        android:layout_marginTop="8dp"
        android:visibility="gone" />
</LinearLayout>
```

- [ ] **Step 3: Create LoginActivity**

```kotlin
// ui/auth/LoginActivity.kt
package com.nschatz.tracker.ui.auth

import android.content.Intent
import android.os.Bundle
import android.view.View
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.local.TrackerDatabase
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.data.repository.AuthRepository
import com.nschatz.tracker.databinding.ActivityLoginBinding
import com.nschatz.tracker.ui.main.MainActivity
import kotlinx.coroutines.launch

class LoginActivity : AppCompatActivity() {

    private lateinit var binding: ActivityLoginBinding
    private lateinit var session: SessionManager
    private lateinit var authRepo: AuthRepository

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityLoginBinding.inflate(layoutInflater)
        setContentView(binding.root)

        session = SessionManager(this)

        if (session.isLoggedIn) {
            goToMain()
            return
        }

        val api = ApiClient(session)
        authRepo = AuthRepository(api, session)

        binding.editServerUrl.setText(session.serverUrl)

        binding.btnLogin.setOnClickListener { doLogin() }
        binding.btnRegister.setOnClickListener {
            startActivity(Intent(this, RegisterActivity::class.java))
        }
    }

    private fun doLogin() {
        val serverUrl = binding.editServerUrl.text.toString().trim()
        val email = binding.editEmail.text.toString().trim()
        val password = binding.editPassword.text.toString()

        if (email.isEmpty() || password.isEmpty()) {
            showError("Email and password are required")
            return
        }

        if (serverUrl.isNotEmpty()) {
            session.serverUrl = serverUrl
        }

        setLoading(true)
        lifecycleScope.launch {
            val result = authRepo.login(email, password)
            setLoading(false)
            result.fold(
                onSuccess = { goToMain() },
                onFailure = { showError(it.message ?: "Login failed") }
            )
        }
    }

    private fun goToMain() {
        startActivity(Intent(this, MainActivity::class.java))
        finish()
    }

    private fun setLoading(loading: Boolean) {
        binding.progress.visibility = if (loading) View.VISIBLE else View.GONE
        binding.btnLogin.isEnabled = !loading
        binding.txtError.visibility = View.GONE
    }

    private fun showError(msg: String) {
        binding.txtError.text = msg
        binding.txtError.visibility = View.VISIBLE
    }
}
```

- [ ] **Step 4: Create RegisterActivity**

```kotlin
// ui/auth/RegisterActivity.kt
package com.nschatz.tracker.ui.auth

import android.content.Intent
import android.os.Bundle
import android.view.View
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.data.repository.AuthRepository
import com.nschatz.tracker.databinding.ActivityRegisterBinding
import com.nschatz.tracker.ui.main.MainActivity
import kotlinx.coroutines.launch

class RegisterActivity : AppCompatActivity() {

    private lateinit var binding: ActivityRegisterBinding
    private lateinit var authRepo: AuthRepository

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityRegisterBinding.inflate(layoutInflater)
        setContentView(binding.root)

        val session = SessionManager(this)
        val api = ApiClient(session)
        authRepo = AuthRepository(api, session)

        binding.btnRegister.setOnClickListener { doRegister() }
    }

    private fun doRegister() {
        val name = binding.editDisplayName.text.toString().trim()
        val email = binding.editEmail.text.toString().trim()
        val password = binding.editPassword.text.toString()
        val inviteCode = binding.editInviteCode.text.toString().trim()

        if (name.isEmpty() || email.isEmpty() || password.isEmpty() || inviteCode.isEmpty()) {
            showError("All fields are required")
            return
        }

        setLoading(true)
        lifecycleScope.launch {
            val result = authRepo.register(email, name, password, inviteCode)
            setLoading(false)
            result.fold(
                onSuccess = {
                    startActivity(Intent(this@RegisterActivity, MainActivity::class.java)
                        .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK))
                    finish()
                },
                onFailure = { showError(it.message ?: "Registration failed") }
            )
        }
    }

    private fun setLoading(loading: Boolean) {
        binding.progress.visibility = if (loading) View.VISIBLE else View.GONE
        binding.btnRegister.isEnabled = !loading
        binding.txtError.visibility = View.GONE
    }

    private fun showError(msg: String) {
        binding.txtError.text = msg
        binding.txtError.visibility = View.VISIBLE
    }
}
```

- [ ] **Step 5: Verify build**

```bash
cd android && ./gradlew compileDebugKotlin
```

- [ ] **Step 6: Commit**

```bash
git add android/
git commit -m "feat: login and register screens with server URL configuration"
```

---

## Task 6: Location Foreground Service

**Files:**
- Create: `android/app/src/main/java/com/nschatz/tracker/service/LocationService.kt`
- Create: `android/app/src/main/java/com/nschatz/tracker/service/ActivityTransitionReceiver.kt`

- [ ] **Step 1: Create LocationService**

```kotlin
// service/LocationService.kt
package com.nschatz.tracker.service

import android.app.*
import android.content.Intent
import android.content.pm.ServiceInfo
import android.os.IBinder
import android.os.Looper
import android.util.Log
import androidx.core.app.NotificationCompat
import com.google.android.gms.location.*
import com.nschatz.tracker.TrackerApp
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.local.TrackerDatabase
import com.nschatz.tracker.data.model.LocationBatch
import com.nschatz.tracker.data.model.LocationPoint
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.data.repository.LocationRepository
import com.nschatz.tracker.ui.main.MainActivity
import kotlinx.coroutines.*
import java.time.Instant

class LocationService : Service() {

    companion object {
        const val TAG = "LocationService"
        const val NOTIFICATION_ID = 1
        const val ACTION_START = "START"
        const val ACTION_STOP = "STOP"
        const val ACTION_UPDATE_INTERVAL = "UPDATE_INTERVAL"
        const val EXTRA_INTERVAL_MS = "interval_ms"

        const val INTERVAL_STATIONARY = 300_000L  // 5 minutes
        const val INTERVAL_WALKING = 30_000L      // 30 seconds
        const val INTERVAL_DRIVING = 10_000L      // 10 seconds
    }

    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private lateinit var fusedClient: FusedLocationProviderClient
    private lateinit var locationRepo: LocationRepository
    private lateinit var session: SessionManager
    private var currentInterval = INTERVAL_STATIONARY

    private val locationCallback = object : LocationCallback() {
        override fun onLocationResult(result: LocationResult) {
            val location = result.lastLocation ?: return
            scope.launch {
                try {
                    val battery = getBatteryLevel()
                    val point = LocationPoint(
                        lat = location.latitude,
                        lng = location.longitude,
                        speed = if (location.hasSpeed()) location.speed else null,
                        batteryLevel = battery,
                        accuracy = if (location.hasAccuracy()) location.accuracy else null,
                        recordedAt = Instant.now().toString()
                    )

                    val result = locationRepo.submitLocations(listOf(point))
                    if (result.isFailure) {
                        locationRepo.queueLocation(
                            point.lat, point.lng, point.speed, point.batteryLevel, point.accuracy
                        )
                        Log.d(TAG, "Queued location (offline)")
                    } else {
                        // Try to flush any queued locations
                        locationRepo.flushQueue()
                    }
                } catch (e: Exception) {
                    Log.e(TAG, "Error processing location", e)
                }
            }
        }
    }

    override fun onCreate() {
        super.onCreate()
        fusedClient = LocationServices.getFusedLocationProviderClient(this)
        session = SessionManager(this)
        val api = ApiClient(session)
        val db = TrackerDatabase.getInstance(this)
        locationRepo = LocationRepository(api, db.pendingLocationDao())
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_STOP -> {
                stopForeground(STOP_FOREGROUND_REMOVE)
                stopSelf()
                return START_NOT_STICKY
            }
            ACTION_UPDATE_INTERVAL -> {
                val interval = intent.getLongExtra(EXTRA_INTERVAL_MS, INTERVAL_STATIONARY)
                updateInterval(interval)
                return START_STICKY
            }
            else -> {
                startForeground(
                    NOTIFICATION_ID,
                    createNotification(),
                    ServiceInfo.FOREGROUND_SERVICE_TYPE_LOCATION
                )
                startLocationUpdates()
                registerActivityTransitions()
                return START_STICKY
            }
        }
    }

    override fun onDestroy() {
        scope.cancel()
        fusedClient.removeLocationUpdates(locationCallback)
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun startLocationUpdates() {
        try {
            val request = LocationRequest.Builder(Priority.PRIORITY_HIGH_ACCURACY, currentInterval)
                .setMinUpdateIntervalMillis(currentInterval / 2)
                .build()

            fusedClient.requestLocationUpdates(request, locationCallback, Looper.getMainLooper())
            Log.d(TAG, "Location updates started (interval: ${currentInterval}ms)")
        } catch (e: SecurityException) {
            Log.e(TAG, "Missing location permission", e)
        }
    }

    private fun updateInterval(intervalMs: Long) {
        if (intervalMs == currentInterval) return
        currentInterval = intervalMs
        fusedClient.removeLocationUpdates(locationCallback)
        startLocationUpdates()
        Log.d(TAG, "Interval updated to ${intervalMs}ms")
    }

    private fun registerActivityTransitions() {
        val transitions = listOf(
            ActivityTransition.Builder()
                .setActivityType(DetectedActivity.STILL)
                .setActivityTransition(ActivityTransition.ACTIVITY_TRANSITION_ENTER)
                .build(),
            ActivityTransition.Builder()
                .setActivityType(DetectedActivity.WALKING)
                .setActivityTransition(ActivityTransition.ACTIVITY_TRANSITION_ENTER)
                .build(),
            ActivityTransition.Builder()
                .setActivityType(DetectedActivity.IN_VEHICLE)
                .setActivityTransition(ActivityTransition.ACTIVITY_TRANSITION_ENTER)
                .build()
        )

        val request = ActivityTransitionRequest(transitions)
        val intent = Intent(this, ActivityTransitionReceiver::class.java)
        val pendingIntent = PendingIntent.getBroadcast(
            this, 0, intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_MUTABLE
        )

        try {
            ActivityRecognition.getClient(this)
                .requestActivityTransitionUpdates(request, pendingIntent)
            Log.d(TAG, "Activity transitions registered")
        } catch (e: SecurityException) {
            Log.e(TAG, "Missing activity recognition permission", e)
        }
    }

    private fun createNotification(): Notification {
        val tapIntent = PendingIntent.getActivity(
            this, 0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE
        )

        return NotificationCompat.Builder(this, TrackerApp.CHANNEL_LOCATION)
            .setContentTitle("Tracker Active")
            .setContentText("Sharing your location")
            .setSmallIcon(android.R.drawable.ic_menu_mylocation)
            .setContentIntent(tapIntent)
            .setOngoing(true)
            .build()
    }

    private fun getBatteryLevel(): Int {
        val batteryManager = getSystemService(BATTERY_SERVICE) as android.os.BatteryManager
        return batteryManager.getIntProperty(android.os.BatteryManager.BATTERY_PROPERTY_CAPACITY)
    }
}
```

- [ ] **Step 2: Create ActivityTransitionReceiver**

```kotlin
// service/ActivityTransitionReceiver.kt
package com.nschatz.tracker.service

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log
import com.google.android.gms.location.ActivityTransitionResult
import com.google.android.gms.location.DetectedActivity

class ActivityTransitionReceiver : BroadcastReceiver() {

    override fun onReceive(context: Context, intent: Intent) {
        if (!ActivityTransitionResult.hasResult(intent)) return

        val result = ActivityTransitionResult.extractResult(intent) ?: return

        for (event in result.transitionEvents) {
            val interval = when (event.activityType) {
                DetectedActivity.STILL -> LocationService.INTERVAL_STATIONARY
                DetectedActivity.WALKING -> LocationService.INTERVAL_WALKING
                DetectedActivity.IN_VEHICLE -> LocationService.INTERVAL_DRIVING
                else -> null
            }

            if (interval != null) {
                Log.d("ActivityTransition", "Activity: ${event.activityType}, interval: ${interval}ms")
                val serviceIntent = Intent(context, LocationService::class.java).apply {
                    action = LocationService.ACTION_UPDATE_INTERVAL
                    putExtra(LocationService.EXTRA_INTERVAL_MS, interval)
                }
                context.startForegroundService(serviceIntent)
            }
        }
    }
}
```

- [ ] **Step 3: Verify build**

```bash
cd android && ./gradlew compileDebugKotlin
```

- [ ] **Step 4: Commit**

```bash
git add android/
git commit -m "feat: location foreground service with adaptive intervals via activity recognition"
```

---

## Task 7: WebSocket Client

**Files:**
- Create: `android/app/src/main/java/com/nschatz/tracker/websocket/LocationWebSocketClient.kt`

- [ ] **Step 1: Create WebSocket client**

```kotlin
// websocket/LocationWebSocketClient.kt
package com.nschatz.tracker.websocket

import android.util.Log
import com.google.gson.Gson
import com.nschatz.tracker.data.model.MemberLocation
import com.nschatz.tracker.data.prefs.SessionManager
import okhttp3.*
import java.util.concurrent.TimeUnit

class LocationWebSocketClient(
    private val session: SessionManager,
    private val onLocationUpdate: (MemberLocation) -> Unit
) {
    companion object {
        const val TAG = "LocationWS"
        const val RECONNECT_DELAY_MS = 5000L
    }

    private val gson = Gson()
    private var webSocket: WebSocket? = null
    private var isConnected = false
    private var shouldReconnect = false

    private val client = OkHttpClient.Builder()
        .readTimeout(0, TimeUnit.MILLISECONDS)
        .pingInterval(30, TimeUnit.SECONDS)
        .build()

    fun connect(circleId: String) {
        shouldReconnect = true
        doConnect(circleId)
    }

    fun disconnect() {
        shouldReconnect = false
        webSocket?.close(1000, "client disconnect")
        webSocket = null
        isConnected = false
    }

    private fun doConnect(circleId: String) {
        val token = session.token ?: return
        val baseUrl = session.serverUrl
            .replace("http://", "ws://")
            .replace("https://", "wss://")
            .trimEnd('/')

        val url = "$baseUrl/ws?circle_id=$circleId"
        val request = Request.Builder()
            .url(url)
            .header("Authorization", "Bearer $token")
            .build()

        webSocket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(ws: WebSocket, response: Response) {
                isConnected = true
                Log.d(TAG, "Connected to $url")
            }

            override fun onMessage(ws: WebSocket, text: String) {
                try {
                    val location = gson.fromJson(text, MemberLocation::class.java)
                    onLocationUpdate(location)
                } catch (e: Exception) {
                    Log.e(TAG, "Parse error: $text", e)
                }
            }

            override fun onClosed(ws: WebSocket, code: Int, reason: String) {
                isConnected = false
                Log.d(TAG, "Closed: $code $reason")
                scheduleReconnect(circleId)
            }

            override fun onFailure(ws: WebSocket, t: Throwable, response: Response?) {
                isConnected = false
                Log.e(TAG, "Connection failed", t)
                scheduleReconnect(circleId)
            }
        })
    }

    private fun scheduleReconnect(circleId: String) {
        if (!shouldReconnect) return
        Thread {
            Thread.sleep(RECONNECT_DELAY_MS)
            if (shouldReconnect) {
                Log.d(TAG, "Reconnecting...")
                doConnect(circleId)
            }
        }.start()
    }
}
```

- [ ] **Step 2: Verify build**

```bash
cd android && ./gradlew compileDebugKotlin
```

- [ ] **Step 3: Commit**

```bash
git add android/
git commit -m "feat: WebSocket client with auto-reconnect for real-time location updates"
```

---

## Task 8: FCM Integration

**Files:**
- Create: `android/app/src/main/java/com/nschatz/tracker/fcm/TrackerFirebaseService.kt`

- [ ] **Step 1: Create Firebase messaging service**

```kotlin
// fcm/TrackerFirebaseService.kt
package com.nschatz.tracker.fcm

import android.app.PendingIntent
import android.content.Intent
import android.util.Log
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.nschatz.tracker.TrackerApp
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.model.FcmTokenRequest
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.ui.main.MainActivity
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

class TrackerFirebaseService : FirebaseMessagingService() {

    override fun onNewToken(token: String) {
        Log.d("FCM", "New token: $token")
        val session = SessionManager(this)
        if (session.isLoggedIn) {
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val api = ApiClient(session)
                    api.fcm.registerToken(FcmTokenRequest(token))
                } catch (e: Exception) {
                    Log.e("FCM", "Failed to register token", e)
                }
            }
        }
    }

    override fun onMessageReceived(message: RemoteMessage) {
        val notification = message.notification ?: return

        val intent = PendingIntent.getActivity(
            this, 0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE
        )

        val builder = NotificationCompat.Builder(this, TrackerApp.CHANNEL_PLACE_ALERTS)
            .setSmallIcon(android.R.drawable.ic_dialog_map)
            .setContentTitle(notification.title)
            .setContentText(notification.body)
            .setContentIntent(intent)
            .setAutoCancel(true)
            .setPriority(NotificationCompat.PRIORITY_HIGH)

        try {
            NotificationManagerCompat.from(this).notify(
                System.currentTimeMillis().toInt(), builder.build()
            )
        } catch (e: SecurityException) {
            Log.e("FCM", "Missing notification permission", e)
        }
    }
}
```

- [ ] **Step 2: Verify build**

```bash
cd android && ./gradlew compileDebugKotlin
```

- [ ] **Step 3: Commit**

```bash
git add android/
git commit -m "feat: FCM service for push notifications and token registration"
```

---

## Task 9: Main Activity + Bottom Navigation

**Files:**
- Create: `android/app/src/main/res/layout/activity_main.xml`
- Create: `android/app/src/main/res/menu/bottom_nav.xml`
- Create: `android/app/src/main/res/drawable/ic_map.xml`
- Create: `android/app/src/main/res/drawable/ic_history.xml`
- Create: `android/app/src/main/res/drawable/ic_place.xml`
- Create: `android/app/src/main/res/drawable/ic_group.xml`
- Create: `android/app/src/main/res/drawable/ic_person.xml`
- Create: `android/app/src/main/java/com/nschatz/tracker/ui/main/MainActivity.kt`

- [ ] **Step 1: Create drawable icons** (Material icons as vector XML)

Create 5 vector drawable XMLs for map, history, place, group, person icons using standard Material paths. These are placeholder vector icons.

- [ ] **Step 2: Create bottom navigation menu**

```xml
<!-- res/menu/bottom_nav.xml -->
<?xml version="1.0" encoding="utf-8"?>
<menu xmlns:android="http://schemas.android.com/apk/res/android">
    <item android:id="@+id/nav_map"
        android:icon="@drawable/ic_map"
        android:title="@string/nav_map" />
    <item android:id="@+id/nav_history"
        android:icon="@drawable/ic_history"
        android:title="@string/nav_history" />
    <item android:id="@+id/nav_places"
        android:icon="@drawable/ic_place"
        android:title="@string/nav_places" />
    <item android:id="@+id/nav_circle"
        android:icon="@drawable/ic_group"
        android:title="@string/nav_circle" />
    <item android:id="@+id/nav_profile"
        android:icon="@drawable/ic_person"
        android:title="@string/nav_profile" />
</menu>
```

- [ ] **Step 3: Create main activity layout**

```xml
<!-- res/layout/activity_main.xml -->
<?xml version="1.0" encoding="utf-8"?>
<LinearLayout xmlns:android="http://schemas.android.com/apk/res/android"
    xmlns:app="http://schemas.android.com/apk/res-auto"
    android:layout_width="match_parent"
    android:layout_height="match_parent"
    android:orientation="vertical">

    <FrameLayout
        android:id="@+id/fragmentContainer"
        android:layout_width="match_parent"
        android:layout_height="0dp"
        android:layout_weight="1" />

    <com.google.android.material.bottomnavigation.BottomNavigationView
        android:id="@+id/bottomNav"
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        app:menu="@menu/bottom_nav" />
</LinearLayout>
```

- [ ] **Step 4: Create MainActivity**

```kotlin
// ui/main/MainActivity.kt
package com.nschatz.tracker.ui.main

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.fragment.app.Fragment
import com.google.firebase.messaging.FirebaseMessaging
import com.nschatz.tracker.R
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.model.FcmTokenRequest
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.databinding.ActivityMainBinding
import com.nschatz.tracker.service.LocationService
import com.nschatz.tracker.ui.circle.CircleFragment
import com.nschatz.tracker.ui.history.HistoryFragment
import com.nschatz.tracker.ui.map.MapFragment
import com.nschatz.tracker.ui.places.PlacesFragment
import com.nschatz.tracker.ui.profile.ProfileFragment
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding
    private lateinit var session: SessionManager

    private val locationPermissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions()
    ) { permissions ->
        if (permissions[Manifest.permission.ACCESS_FINE_LOCATION] == true) {
            requestBackgroundLocation()
        }
    }

    private val backgroundLocationLauncher = registerForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) { granted ->
        if (granted) startLocationService()
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)

        session = SessionManager(this)

        binding.bottomNav.setOnItemSelectedListener { item ->
            val fragment: Fragment = when (item.itemId) {
                R.id.nav_map -> MapFragment()
                R.id.nav_history -> HistoryFragment()
                R.id.nav_places -> PlacesFragment()
                R.id.nav_circle -> CircleFragment()
                R.id.nav_profile -> ProfileFragment()
                else -> return@setOnItemSelectedListener false
            }
            supportFragmentManager.beginTransaction()
                .replace(R.id.fragmentContainer, fragment)
                .commit()
            true
        }

        if (savedInstanceState == null) {
            binding.bottomNav.selectedItemId = R.id.nav_map
        }

        requestPermissions()
        registerFcmToken()
    }

    private fun requestPermissions() {
        val needed = mutableListOf<String>()
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.ACCESS_FINE_LOCATION)
            != PackageManager.PERMISSION_GRANTED) {
            needed.add(Manifest.permission.ACCESS_FINE_LOCATION)
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS)
            != PackageManager.PERMISSION_GRANTED) {
            needed.add(Manifest.permission.POST_NOTIFICATIONS)
        }
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.ACTIVITY_RECOGNITION)
            != PackageManager.PERMISSION_GRANTED) {
            needed.add(Manifest.permission.ACTIVITY_RECOGNITION)
        }

        if (needed.isNotEmpty()) {
            locationPermissionLauncher.launch(needed.toTypedArray())
        } else {
            requestBackgroundLocation()
        }
    }

    private fun requestBackgroundLocation() {
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.ACCESS_BACKGROUND_LOCATION)
            != PackageManager.PERMISSION_GRANTED) {
            backgroundLocationLauncher.launch(Manifest.permission.ACCESS_BACKGROUND_LOCATION)
        } else {
            startLocationService()
        }
    }

    private fun startLocationService() {
        val intent = Intent(this, LocationService::class.java).apply {
            action = LocationService.ACTION_START
        }
        startForegroundService(intent)
    }

    private fun registerFcmToken() {
        FirebaseMessaging.getInstance().token.addOnSuccessListener { token ->
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val api = ApiClient(session)
                    api.fcm.registerToken(FcmTokenRequest(token))
                } catch (e: Exception) {
                    // Will retry on next app launch
                }
            }
        }
    }
}
```

- [ ] **Step 5: Verify build**

```bash
cd android && ./gradlew compileDebugKotlin
```

- [ ] **Step 6: Commit**

```bash
git add android/
git commit -m "feat: main activity with bottom navigation and permission handling"
```

---

## Task 10: Map Fragment

**Files:**
- Create: `android/app/src/main/res/layout/fragment_map.xml`
- Create: `android/app/src/main/java/com/nschatz/tracker/ui/map/MapFragment.kt`

- [ ] **Step 1: Create map layout**

```xml
<!-- res/layout/fragment_map.xml -->
<?xml version="1.0" encoding="utf-8"?>
<FrameLayout xmlns:android="http://schemas.android.com/apk/res/android"
    android:layout_width="match_parent"
    android:layout_height="match_parent">

    <org.osmdroid.views.MapView
        android:id="@+id/mapView"
        android:layout_width="match_parent"
        android:layout_height="match_parent" />

    <ProgressBar
        android:id="@+id/progress"
        android:layout_width="wrap_content"
        android:layout_height="wrap_content"
        android:layout_gravity="center"
        android:visibility="gone" />
</FrameLayout>
```

- [ ] **Step 2: Create MapFragment**

```kotlin
// ui/map/MapFragment.kt
package com.nschatz.tracker.ui.map

import android.graphics.Color
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.local.TrackerDatabase
import com.nschatz.tracker.data.model.Geofence
import com.nschatz.tracker.data.model.MemberLocation
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.data.repository.CircleRepository
import com.nschatz.tracker.data.repository.GeofenceRepository
import com.nschatz.tracker.data.repository.LocationRepository
import com.nschatz.tracker.databinding.FragmentMapBinding
import com.nschatz.tracker.websocket.LocationWebSocketClient
import kotlinx.coroutines.launch
import org.osmdroid.tileprovider.tilesource.TileSourceFactory
import org.osmdroid.util.GeoPoint
import org.osmdroid.views.overlay.Marker
import org.osmdroid.views.overlay.Polygon
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

class MapFragment : Fragment() {

    private var _binding: FragmentMapBinding? = null
    private val binding get() = _binding!!

    private lateinit var session: SessionManager
    private lateinit var locationRepo: LocationRepository
    private lateinit var circleRepo: CircleRepository
    private lateinit var geofenceRepo: GeofenceRepository
    private var wsClient: LocationWebSocketClient? = null

    private val memberMarkers = mutableMapOf<String, Marker>()
    private val geofenceOverlays = mutableListOf<Polygon>()
    private var members: Map<String, String> = emptyMap() // userId -> displayName

    private val timeFormatter = DateTimeFormatter.ofPattern("h:mm a")
        .withZone(ZoneId.systemDefault())

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        _binding = FragmentMapBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        session = SessionManager(requireContext())
        val api = ApiClient(session)
        val db = TrackerDatabase.getInstance(requireContext())
        locationRepo = LocationRepository(api, db.pendingLocationDao())
        circleRepo = CircleRepository(api)
        geofenceRepo = GeofenceRepository(api)

        binding.mapView.setTileSource(TileSourceFactory.MAPNIK)
        binding.mapView.setMultiTouchControls(true)
        binding.mapView.controller.setZoom(15.0)

        loadData()
    }

    override fun onResume() {
        super.onResume()
        binding.mapView.onResume()
        connectWebSocket()
    }

    override fun onPause() {
        binding.mapView.onPause()
        wsClient?.disconnect()
        super.onPause()
    }

    override fun onDestroyView() {
        _binding = null
        super.onDestroyView()
    }

    private fun loadData() {
        val circleId = session.activeCircleId ?: return

        lifecycleScope.launch {
            // Load members for display names
            circleRepo.getMembers(circleId).onSuccess { memberList ->
                members = memberList.associate { it.userId to it.displayName }
            }

            // Load latest positions
            locationRepo.getLatest(circleId).onSuccess { locations ->
                updateMarkers(locations)
                if (locations.isNotEmpty()) {
                    val first = locations.first()
                    binding.mapView.controller.setCenter(GeoPoint(first.lat, first.lng))
                }
            }

            // Load geofences
            geofenceRepo.getAll(circleId).onSuccess { geofences ->
                drawGeofences(geofences)
            }
        }
    }

    private fun connectWebSocket() {
        val circleId = session.activeCircleId ?: return
        wsClient = LocationWebSocketClient(session) { location ->
            requireActivity().runOnUiThread {
                updateMarker(location)
            }
        }
        wsClient?.connect(circleId)
    }

    private fun updateMarkers(locations: List<MemberLocation>) {
        for (loc in locations) {
            updateMarker(loc)
        }
    }

    private fun updateMarker(loc: MemberLocation) {
        val point = GeoPoint(loc.lat, loc.lng)
        val displayName = members[loc.userId] ?: loc.userId.take(8)
        val time = try {
            timeFormatter.format(Instant.parse(loc.recordedAt))
        } catch (e: Exception) { loc.recordedAt }

        val marker = memberMarkers.getOrPut(loc.userId) {
            Marker(binding.mapView).also {
                binding.mapView.overlays.add(it)
            }
        }

        marker.position = point
        marker.title = displayName
        marker.snippet = buildString {
            append(time)
            if (loc.batteryLevel != null) append(" · ${loc.batteryLevel}%")
        }
        marker.setAnchor(Marker.ANCHOR_CENTER, Marker.ANCHOR_BOTTOM)

        binding.mapView.invalidate()
    }

    private fun drawGeofences(geofences: List<Geofence>) {
        geofenceOverlays.forEach { binding.mapView.overlays.remove(it) }
        geofenceOverlays.clear()

        for (gf in geofences) {
            val circle = Polygon(binding.mapView).apply {
                points = Polygon.pointsAsCircle(
                    GeoPoint(gf.lat, gf.lng), gf.radiusMeters.toDouble()
                )
                fillPaint.color = Color.parseColor("#331B73E8")
                outlinePaint.color = Color.parseColor("#881B73E8")
                outlinePaint.strokeWidth = 3f
                title = gf.name
            }
            geofenceOverlays.add(circle)
            binding.mapView.overlays.add(circle)
        }
        binding.mapView.invalidate()
    }
}
```

- [ ] **Step 3: Verify build**

```bash
cd android && ./gradlew compileDebugKotlin
```

- [ ] **Step 4: Commit**

```bash
git add android/
git commit -m "feat: live map with member markers, geofence circles, and WebSocket updates"
```

---

## Task 11: History Fragment

**Files:**
- Create: `android/app/src/main/res/layout/fragment_history.xml`
- Create: `android/app/src/main/java/com/nschatz/tracker/ui/history/HistoryFragment.kt`

- [ ] **Step 1: Create history layout**

```xml
<!-- res/layout/fragment_history.xml -->
<?xml version="1.0" encoding="utf-8"?>
<LinearLayout xmlns:android="http://schemas.android.com/apk/res/android"
    android:layout_width="match_parent"
    android:layout_height="match_parent"
    android:orientation="vertical">

    <LinearLayout
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:orientation="horizontal"
        android:padding="8dp">

        <Spinner
            android:id="@+id/spinnerMember"
            android:layout_width="0dp"
            android:layout_height="wrap_content"
            android:layout_weight="1" />

        <com.google.android.material.button.MaterialButton
            android:id="@+id/btnDate"
            android:layout_width="wrap_content"
            android:layout_height="wrap_content"
            android:text="Today"
            style="@style/Widget.Material3.Button.OutlinedButton" />
    </LinearLayout>

    <org.osmdroid.views.MapView
        android:id="@+id/mapView"
        android:layout_width="match_parent"
        android:layout_height="0dp"
        android:layout_weight="1" />
</LinearLayout>
```

- [ ] **Step 2: Create HistoryFragment**

```kotlin
// ui/history/HistoryFragment.kt
package com.nschatz.tracker.ui.history

import android.app.DatePickerDialog
import android.graphics.Color
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.local.TrackerDatabase
import com.nschatz.tracker.data.model.CircleMember
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.data.repository.CircleRepository
import com.nschatz.tracker.data.repository.LocationRepository
import com.nschatz.tracker.databinding.FragmentHistoryBinding
import kotlinx.coroutines.launch
import org.osmdroid.tileprovider.tilesource.TileSourceFactory
import org.osmdroid.util.GeoPoint
import org.osmdroid.views.overlay.Polyline
import java.time.*
import java.time.format.DateTimeFormatter

class HistoryFragment : Fragment() {

    private var _binding: FragmentHistoryBinding? = null
    private val binding get() = _binding!!

    private lateinit var session: SessionManager
    private lateinit var locationRepo: LocationRepository
    private lateinit var circleRepo: CircleRepository
    private var memberList: List<CircleMember> = emptyList()
    private var selectedDate: LocalDate = LocalDate.now()
    private var pathOverlay: Polyline? = null

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        _binding = FragmentHistoryBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        session = SessionManager(requireContext())
        val api = ApiClient(session)
        val db = TrackerDatabase.getInstance(requireContext())
        locationRepo = LocationRepository(api, db.pendingLocationDao())
        circleRepo = CircleRepository(api)

        binding.mapView.setTileSource(TileSourceFactory.MAPNIK)
        binding.mapView.setMultiTouchControls(true)
        binding.mapView.controller.setZoom(14.0)

        binding.btnDate.setOnClickListener { showDatePicker() }

        binding.spinnerMember.setOnItemSelectedListener(object :
            android.widget.AdapterView.OnItemSelectedListener {
            override fun onItemSelected(parent: android.widget.AdapterView<*>?, v: View?, pos: Int, id: Long) {
                loadHistory()
            }
            override fun onNothingSelected(parent: android.widget.AdapterView<*>?) {}
        })

        loadMembers()
    }

    override fun onResume() { super.onResume(); binding.mapView.onResume() }
    override fun onPause() { binding.mapView.onPause(); super.onPause() }
    override fun onDestroyView() { _binding = null; super.onDestroyView() }

    private fun loadMembers() {
        val circleId = session.activeCircleId ?: return
        lifecycleScope.launch {
            circleRepo.getMembers(circleId).onSuccess { members ->
                memberList = members
                val names = members.map { it.displayName.ifEmpty { it.email } }
                binding.spinnerMember.adapter = ArrayAdapter(
                    requireContext(), android.R.layout.simple_spinner_dropdown_item, names
                )
            }
        }
    }

    private fun showDatePicker() {
        DatePickerDialog(requireContext(), { _, year, month, day ->
            selectedDate = LocalDate.of(year, month + 1, day)
            binding.btnDate.text = selectedDate.format(DateTimeFormatter.ofPattern("MMM d"))
            loadHistory()
        }, selectedDate.year, selectedDate.monthValue - 1, selectedDate.dayOfMonth).show()
    }

    private fun loadHistory() {
        val pos = binding.spinnerMember.selectedItemPosition
        if (pos < 0 || pos >= memberList.size) return
        val member = memberList[pos]

        val from = selectedDate.atStartOfDay(ZoneId.systemDefault()).toInstant()
        val to = selectedDate.plusDays(1).atStartOfDay(ZoneId.systemDefault()).toInstant()

        lifecycleScope.launch {
            locationRepo.getHistory(member.userId, from, to).onSuccess { locations ->
                pathOverlay?.let { binding.mapView.overlays.remove(it) }

                if (locations.isEmpty()) return@onSuccess

                val points = locations.map { GeoPoint(it.lat, it.lng) }
                pathOverlay = Polyline(binding.mapView).apply {
                    setPoints(points)
                    outlinePaint.color = Color.parseColor("#1B73E8")
                    outlinePaint.strokeWidth = 6f
                }
                binding.mapView.overlays.add(pathOverlay)
                binding.mapView.controller.setCenter(points[points.size / 2])
                binding.mapView.invalidate()
            }
        }
    }
}
```

- [ ] **Step 3: Verify build**

```bash
cd android && ./gradlew compileDebugKotlin
```

- [ ] **Step 4: Commit**

```bash
git add android/
git commit -m "feat: history screen with member selection, date picker, and path overlay"
```

---

## Task 12: Places Fragment (Geofence Management)

**Files:**
- Create: `android/app/src/main/res/layout/fragment_places.xml`
- Create: `android/app/src/main/res/layout/item_geofence.xml`
- Create: `android/app/src/main/res/layout/dialog_geofence_edit.xml`
- Create: `android/app/src/main/java/com/nschatz/tracker/ui/places/PlacesFragment.kt`

- [ ] **Step 1: Create places layout**

```xml
<!-- res/layout/fragment_places.xml -->
<?xml version="1.0" encoding="utf-8"?>
<LinearLayout xmlns:android="http://schemas.android.com/apk/res/android"
    android:layout_width="match_parent"
    android:layout_height="match_parent"
    android:orientation="vertical">

    <androidx.recyclerview.widget.RecyclerView
        android:id="@+id/recyclerGeofences"
        android:layout_width="match_parent"
        android:layout_height="0dp"
        android:layout_weight="1" />

    <com.google.android.material.floatingactionbutton.FloatingActionButton
        android:id="@+id/fabAdd"
        android:layout_width="wrap_content"
        android:layout_height="wrap_content"
        android:layout_gravity="end|bottom"
        android:layout_margin="16dp"
        android:src="@android:drawable/ic_input_add"
        android:contentDescription="Add place" />
</LinearLayout>
```

```xml
<!-- res/layout/item_geofence.xml -->
<?xml version="1.0" encoding="utf-8"?>
<LinearLayout xmlns:android="http://schemas.android.com/apk/res/android"
    android:layout_width="match_parent"
    android:layout_height="wrap_content"
    android:orientation="horizontal"
    android:padding="16dp"
    android:gravity="center_vertical">

    <LinearLayout
        android:layout_width="0dp"
        android:layout_height="wrap_content"
        android:layout_weight="1"
        android:orientation="vertical">

        <TextView
            android:id="@+id/txtName"
            android:layout_width="wrap_content"
            android:layout_height="wrap_content"
            android:textSize="16sp"
            android:textStyle="bold" />

        <TextView
            android:id="@+id/txtRadius"
            android:layout_width="wrap_content"
            android:layout_height="wrap_content"
            android:textSize="14sp"
            android:textColor="#757575" />
    </LinearLayout>

    <ImageButton
        android:id="@+id/btnDelete"
        android:layout_width="40dp"
        android:layout_height="40dp"
        android:src="@android:drawable/ic_menu_delete"
        android:background="?attr/selectableItemBackgroundBorderless"
        android:contentDescription="Delete" />
</LinearLayout>
```

```xml
<!-- res/layout/dialog_geofence_edit.xml -->
<?xml version="1.0" encoding="utf-8"?>
<LinearLayout xmlns:android="http://schemas.android.com/apk/res/android"
    android:layout_width="match_parent"
    android:layout_height="wrap_content"
    android:orientation="vertical"
    android:padding="24dp">

    <com.google.android.material.textfield.TextInputLayout
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:layout_marginBottom="8dp"
        android:hint="Place name">
        <com.google.android.material.textfield.TextInputEditText
            android:id="@+id/editName"
            android:layout_width="match_parent"
            android:layout_height="wrap_content" />
    </com.google.android.material.textfield.TextInputLayout>

    <com.google.android.material.textfield.TextInputLayout
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:layout_marginBottom="8dp"
        android:hint="Radius (meters)">
        <com.google.android.material.textfield.TextInputEditText
            android:id="@+id/editRadius"
            android:layout_width="match_parent"
            android:layout_height="wrap_content"
            android:inputType="number"
            android:text="100" />
    </com.google.android.material.textfield.TextInputLayout>

    <TextView
        android:layout_width="wrap_content"
        android:layout_height="wrap_content"
        android:text="Tap the map to set the center point"
        android:textColor="#757575"
        android:layout_marginBottom="8dp" />

    <org.osmdroid.views.MapView
        android:id="@+id/mapPicker"
        android:layout_width="match_parent"
        android:layout_height="250dp" />
</LinearLayout>
```

- [ ] **Step 2: Create PlacesFragment**

```kotlin
// ui/places/PlacesFragment.kt
package com.nschatz.tracker.ui.places

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Toast
import androidx.appcompat.app.AlertDialog
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.model.Geofence
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.data.repository.GeofenceRepository
import com.nschatz.tracker.databinding.FragmentPlacesBinding
import com.nschatz.tracker.databinding.ItemGeofenceBinding
import com.nschatz.tracker.databinding.DialogGeofenceEditBinding
import kotlinx.coroutines.launch
import org.osmdroid.events.MapEventsReceiver
import org.osmdroid.tileprovider.tilesource.TileSourceFactory
import org.osmdroid.util.GeoPoint
import org.osmdroid.views.overlay.MapEventsOverlay
import org.osmdroid.views.overlay.Marker

class PlacesFragment : Fragment() {

    private var _binding: FragmentPlacesBinding? = null
    private val binding get() = _binding!!
    private lateinit var session: SessionManager
    private lateinit var geofenceRepo: GeofenceRepository
    private val geofences = mutableListOf<Geofence>()

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        _binding = FragmentPlacesBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        session = SessionManager(requireContext())
        val api = ApiClient(session)
        geofenceRepo = GeofenceRepository(api)

        binding.recyclerGeofences.layoutManager = LinearLayoutManager(requireContext())
        binding.recyclerGeofences.adapter = GeofenceAdapter()

        binding.fabAdd.setOnClickListener { showCreateDialog() }

        loadGeofences()
    }

    override fun onDestroyView() { _binding = null; super.onDestroyView() }

    private fun loadGeofences() {
        val circleId = session.activeCircleId ?: return
        lifecycleScope.launch {
            geofenceRepo.getAll(circleId).onSuccess {
                geofences.clear()
                geofences.addAll(it)
                binding.recyclerGeofences.adapter?.notifyDataSetChanged()
            }
        }
    }

    private fun showCreateDialog() {
        val dialogBinding = DialogGeofenceEditBinding.inflate(layoutInflater)
        var selectedPoint: GeoPoint? = null
        var marker: Marker? = null

        dialogBinding.mapPicker.setTileSource(TileSourceFactory.MAPNIK)
        dialogBinding.mapPicker.setMultiTouchControls(true)
        dialogBinding.mapPicker.controller.setZoom(15.0)

        val eventsOverlay = MapEventsOverlay(object : MapEventsReceiver {
            override fun singleTapConfirmedHelper(p: GeoPoint): Boolean {
                selectedPoint = p
                marker?.let { dialogBinding.mapPicker.overlays.remove(it) }
                marker = Marker(dialogBinding.mapPicker).apply {
                    position = p
                    setAnchor(Marker.ANCHOR_CENTER, Marker.ANCHOR_BOTTOM)
                }
                dialogBinding.mapPicker.overlays.add(marker)
                dialogBinding.mapPicker.invalidate()
                return true
            }
            override fun longPressHelper(p: GeoPoint): Boolean = false
        })
        dialogBinding.mapPicker.overlays.add(eventsOverlay)

        AlertDialog.Builder(requireContext())
            .setTitle("New Place")
            .setView(dialogBinding.root)
            .setPositiveButton("Create") { _, _ ->
                val name = dialogBinding.editName.text.toString().trim()
                val radius = dialogBinding.editRadius.text.toString().toFloatOrNull() ?: 100f
                val point = selectedPoint

                if (name.isEmpty() || point == null) {
                    Toast.makeText(requireContext(), "Name and location required", Toast.LENGTH_SHORT).show()
                    return@setPositiveButton
                }

                val circleId = session.activeCircleId ?: return@setPositiveButton
                lifecycleScope.launch {
                    geofenceRepo.create(circleId, name, point.latitude, point.longitude, radius)
                        .onSuccess { loadGeofences() }
                        .onFailure { Toast.makeText(requireContext(), "Failed to create", Toast.LENGTH_SHORT).show() }
                }
            }
            .setNegativeButton("Cancel", null)
            .show()
    }

    private inner class GeofenceAdapter : RecyclerView.Adapter<GeofenceAdapter.VH>() {
        inner class VH(val binding: ItemGeofenceBinding) : RecyclerView.ViewHolder(binding.root)

        override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): VH {
            return VH(ItemGeofenceBinding.inflate(layoutInflater, parent, false))
        }

        override fun onBindViewHolder(holder: VH, position: Int) {
            val gf = geofences[position]
            holder.binding.txtName.text = gf.name
            holder.binding.txtRadius.text = "${gf.radiusMeters.toInt()}m radius"
            holder.binding.btnDelete.setOnClickListener {
                lifecycleScope.launch {
                    geofenceRepo.delete(gf.id).onSuccess { loadGeofences() }
                }
            }
        }

        override fun getItemCount() = geofences.size
    }
}
```

- [ ] **Step 3: Verify build**

```bash
cd android && ./gradlew compileDebugKotlin
```

- [ ] **Step 4: Commit**

```bash
git add android/
git commit -m "feat: places screen with geofence list, create dialog with map picker"
```

---

## Task 13: Circle Fragment

**Files:**
- Create: `android/app/src/main/res/layout/fragment_circle.xml`
- Create: `android/app/src/main/res/layout/item_member.xml`
- Create: `android/app/src/main/java/com/nschatz/tracker/ui/circle/CircleFragment.kt`

- [ ] **Step 1: Create circle layout and member item**

```xml
<!-- res/layout/fragment_circle.xml -->
<?xml version="1.0" encoding="utf-8"?>
<LinearLayout xmlns:android="http://schemas.android.com/apk/res/android"
    android:layout_width="match_parent"
    android:layout_height="match_parent"
    android:orientation="vertical"
    android:padding="16dp">

    <TextView
        android:id="@+id/txtCircleName"
        android:layout_width="wrap_content"
        android:layout_height="wrap_content"
        android:textSize="24sp"
        android:textStyle="bold"
        android:layout_marginBottom="8dp" />

    <LinearLayout
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:orientation="horizontal"
        android:layout_marginBottom="16dp">

        <TextView
            android:layout_width="wrap_content"
            android:layout_height="wrap_content"
            android:text="Invite code: "
            android:textColor="#757575" />

        <TextView
            android:id="@+id/txtInviteCode"
            android:layout_width="wrap_content"
            android:layout_height="wrap_content"
            android:textStyle="bold" />

        <com.google.android.material.button.MaterialButton
            android:id="@+id/btnCopyCode"
            android:layout_width="wrap_content"
            android:layout_height="wrap_content"
            android:text="Copy"
            android:layout_marginStart="8dp"
            style="@style/Widget.Material3.Button.TextButton" />
    </LinearLayout>

    <TextView
        android:layout_width="wrap_content"
        android:layout_height="wrap_content"
        android:text="Members"
        android:textSize="18sp"
        android:textStyle="bold"
        android:layout_marginBottom="8dp" />

    <androidx.recyclerview.widget.RecyclerView
        android:id="@+id/recyclerMembers"
        android:layout_width="match_parent"
        android:layout_height="0dp"
        android:layout_weight="1" />
</LinearLayout>
```

```xml
<!-- res/layout/item_member.xml -->
<?xml version="1.0" encoding="utf-8"?>
<LinearLayout xmlns:android="http://schemas.android.com/apk/res/android"
    android:layout_width="match_parent"
    android:layout_height="wrap_content"
    android:orientation="vertical"
    android:padding="12dp">

    <TextView
        android:id="@+id/txtMemberName"
        android:layout_width="wrap_content"
        android:layout_height="wrap_content"
        android:textSize="16sp" />

    <TextView
        android:id="@+id/txtMemberRole"
        android:layout_width="wrap_content"
        android:layout_height="wrap_content"
        android:textSize="14sp"
        android:textColor="#757575" />
</LinearLayout>
```

- [ ] **Step 2: Create CircleFragment**

```kotlin
// ui/circle/CircleFragment.kt
package com.nschatz.tracker.ui.circle

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.model.CircleMember
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.data.repository.CircleRepository
import com.nschatz.tracker.databinding.FragmentCircleBinding
import com.nschatz.tracker.databinding.ItemMemberBinding
import kotlinx.coroutines.launch

class CircleFragment : Fragment() {

    private var _binding: FragmentCircleBinding? = null
    private val binding get() = _binding!!
    private lateinit var session: SessionManager
    private lateinit var circleRepo: CircleRepository
    private val members = mutableListOf<CircleMember>()

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        _binding = FragmentCircleBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        session = SessionManager(requireContext())
        circleRepo = CircleRepository(ApiClient(session))

        binding.recyclerMembers.layoutManager = LinearLayoutManager(requireContext())
        binding.recyclerMembers.adapter = MemberAdapter()

        binding.btnCopyCode.setOnClickListener {
            val code = binding.txtInviteCode.text.toString()
            val clipboard = requireContext().getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
            clipboard.setPrimaryClip(ClipData.newPlainText("invite code", code))
            Toast.makeText(requireContext(), "Copied!", Toast.LENGTH_SHORT).show()
        }

        loadCircle()
    }

    override fun onDestroyView() { _binding = null; super.onDestroyView() }

    private fun loadCircle() {
        lifecycleScope.launch {
            circleRepo.getAll().onSuccess { circles ->
                val circle = circles.firstOrNull() ?: return@onSuccess
                session.activeCircleId = circle.id
                binding.txtCircleName.text = circle.name
                binding.txtInviteCode.text = circle.inviteCode

                circleRepo.getMembers(circle.id).onSuccess { memberList ->
                    members.clear()
                    members.addAll(memberList)
                    binding.recyclerMembers.adapter?.notifyDataSetChanged()
                }
            }
        }
    }

    private inner class MemberAdapter : RecyclerView.Adapter<MemberAdapter.VH>() {
        inner class VH(val binding: ItemMemberBinding) : RecyclerView.ViewHolder(binding.root)

        override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): VH {
            return VH(ItemMemberBinding.inflate(layoutInflater, parent, false))
        }

        override fun onBindViewHolder(holder: VH, position: Int) {
            val m = members[position]
            holder.binding.txtMemberName.text = m.displayName.ifEmpty { m.email }
            holder.binding.txtMemberRole.text = m.role
        }

        override fun getItemCount() = members.size
    }
}
```

- [ ] **Step 3: Verify build**

```bash
cd android && ./gradlew compileDebugKotlin
```

- [ ] **Step 4: Commit**

```bash
git add android/
git commit -m "feat: circle screen with member list and invite code sharing"
```

---

## Task 14: Profile Fragment

**Files:**
- Create: `android/app/src/main/res/layout/fragment_profile.xml`
- Create: `android/app/src/main/java/com/nschatz/tracker/ui/profile/ProfileFragment.kt`

- [ ] **Step 1: Create profile layout**

```xml
<!-- res/layout/fragment_profile.xml -->
<?xml version="1.0" encoding="utf-8"?>
<LinearLayout xmlns:android="http://schemas.android.com/apk/res/android"
    android:layout_width="match_parent"
    android:layout_height="match_parent"
    android:orientation="vertical"
    android:padding="24dp">

    <TextView
        android:id="@+id/txtDisplayName"
        android:layout_width="wrap_content"
        android:layout_height="wrap_content"
        android:textSize="24sp"
        android:textStyle="bold"
        android:layout_marginBottom="4dp" />

    <TextView
        android:id="@+id/txtEmail"
        android:layout_width="wrap_content"
        android:layout_height="wrap_content"
        android:textSize="16sp"
        android:textColor="#757575"
        android:layout_marginBottom="24dp" />

    <com.google.android.material.textfield.TextInputLayout
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:layout_marginBottom="16dp"
        android:hint="Server URL">
        <com.google.android.material.textfield.TextInputEditText
            android:id="@+id/editServerUrl"
            android:layout_width="match_parent"
            android:layout_height="wrap_content"
            android:inputType="textUri" />
    </com.google.android.material.textfield.TextInputLayout>

    <com.google.android.material.button.MaterialButton
        android:id="@+id/btnSave"
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:text="Save Settings"
        android:layout_marginBottom="24dp" />

    <com.google.android.material.button.MaterialButton
        android:id="@+id/btnLogout"
        android:layout_width="match_parent"
        android:layout_height="wrap_content"
        android:text="Logout"
        style="@style/Widget.Material3.Button.OutlinedButton" />
</LinearLayout>
```

- [ ] **Step 2: Create ProfileFragment**

```kotlin
// ui/profile/ProfileFragment.kt
package com.nschatz.tracker.ui.profile

import android.content.Intent
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Toast
import androidx.fragment.app.Fragment
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.databinding.FragmentProfileBinding
import com.nschatz.tracker.service.LocationService
import com.nschatz.tracker.ui.auth.LoginActivity

class ProfileFragment : Fragment() {

    private var _binding: FragmentProfileBinding? = null
    private val binding get() = _binding!!
    private lateinit var session: SessionManager

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        _binding = FragmentProfileBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        session = SessionManager(requireContext())

        binding.txtDisplayName.text = session.displayName ?: "Unknown"
        binding.txtEmail.text = session.email ?: ""
        binding.editServerUrl.setText(session.serverUrl)

        binding.btnSave.setOnClickListener {
            val url = binding.editServerUrl.text.toString().trim()
            if (url.isNotEmpty()) {
                session.serverUrl = url
                Toast.makeText(requireContext(), "Saved", Toast.LENGTH_SHORT).show()
            }
        }

        binding.btnLogout.setOnClickListener {
            // Stop location service
            requireContext().startService(
                Intent(requireContext(), LocationService::class.java).apply {
                    action = LocationService.ACTION_STOP
                }
            )
            session.clear()
            startActivity(Intent(requireContext(), LoginActivity::class.java)
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK))
        }
    }

    override fun onDestroyView() { _binding = null; super.onDestroyView() }
}
```

- [ ] **Step 3: Verify build**

```bash
cd android && ./gradlew compileDebugKotlin
```

- [ ] **Step 4: Commit**

```bash
git add android/
git commit -m "feat: profile screen with server URL config and logout"
```

---

## Task 15: Active Circle Initialization

**Files:**
- Modify: `android/app/src/main/java/com/nschatz/tracker/ui/main/MainActivity.kt`

The app needs to set `activeCircleId` on startup so the map and other fragments know which circle to load. When the user first logs in, they may not have a circle ID stored yet.

- [ ] **Step 1: Add circle initialization to MainActivity**

After `requestPermissions()` and `registerFcmToken()` in `onCreate`, add:

```kotlin
initActiveCircle()
```

Add method:
```kotlin
private fun initActiveCircle() {
    if (session.activeCircleId != null) return

    lifecycleScope.launch {
        try {
            val api = ApiClient(session)
            val circleRepo = CircleRepository(api)
            circleRepo.getAll().onSuccess { circles ->
                if (circles.isNotEmpty()) {
                    session.activeCircleId = circles.first().id
                }
            }
        } catch (e: Exception) {
            // Will retry next time
        }
    }
}
```

Add missing imports: `CircleRepository`, `ApiClient`, `lifecycleScope`, `launch`.

- [ ] **Step 2: Verify build**

```bash
cd android && ./gradlew compileDebugKotlin
```

- [ ] **Step 3: Commit**

```bash
git add android/
git commit -m "feat: auto-initialize active circle on first login"
```

---

## Summary

| Task | Component | What it builds |
|------|-----------|---------------|
| 1 | Scaffolding | Android project, Gradle, manifest, notification channels |
| 2 | API Layer | Data models, Retrofit client, token interceptor, session manager |
| 3 | Local Storage | Room database with offline location queue and cached positions |
| 4 | Repositories | Auth, Location, Circle, Geofence repositories |
| 5 | Auth | Login and Register screens with server URL config |
| 6 | Location | Foreground service with adaptive intervals via Activity Recognition |
| 7 | WebSocket | Real-time location client with auto-reconnect |
| 8 | FCM | Push notification handler and token registration |
| 9 | Navigation | MainActivity with bottom nav and permission handling |
| 10 | Map | Live map with member markers, geofences, WebSocket updates |
| 11 | History | Path overlay with member selector and date picker |
| 12 | Places | Geofence list, create/delete with map picker |
| 13 | Circle | Member list and invite code sharing |
| 14 | Profile | Server URL config and logout |
| 15 | Init | Auto-initialize active circle on startup |
