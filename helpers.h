#ifndef WEBVIEW_H
#define WEBVIEW_H

#include <gtk-3.0/gtk/gtk.h>
#include <webkit2/webkit2.h>
#include <JavaScriptCore/JavaScript.h>

extern void closeHandler(guint64);
extern void startHandler(guint64);
extern void wvLoadFinished(guint64, char *);
extern void inGtkMain(guint64);
extern void jsCallback(guint64, JSGlobalContextRef, JSValueRef, char *);

static gboolean window_close_cb(GtkWidget *widget, GdkEvent *event, gpointer parent) {
	(void)widget; (void)event;

	closeHandler((guint64)parent);
	return TRUE;
}

static gboolean wv_context_menu_cb(WebKitWebView *webview,
						GtkWidget *default_menu,
						WebKitHitTestResult *hit_test_result,
						gboolean triggered_with_keyboard,
						gpointer userdata) {
	(void)webview;
	(void)default_menu;
	(void)hit_test_result;
	(void)triggered_with_keyboard;
	(void)userdata;

	return TRUE;
}

static void idle_add(guint64 v) { g_idle_add((GSourceFunc)inGtkMain, (gpointer)v); }
static void timeout_add(guint64 v) { g_timeout_add(100, (GSourceFunc)inGtkMain, (gpointer)v); }

static void wv_load_changed_cb(WebKitWebView *wv, WebKitLoadEvent load_event, gpointer parent) {
	switch (load_event) {
		case WEBKIT_LOAD_STARTED:
			break;
		case WEBKIT_LOAD_REDIRECTED:
			break;
		case WEBKIT_LOAD_COMMITTED:
			break;
		case WEBKIT_LOAD_FINISHED:
			wvLoadFinished((guint64) parent, (char *)webkit_web_view_get_uri(wv));
			break;
		}
}

typedef struct {
	gboolean EnableJava;
	gboolean EnablePlugins;
	gboolean EnableFrameFlattening;
	gboolean EnableSmoothScrolling;

	gboolean EnableJavaScript;
	gboolean EnableJavaScriptCanOpenWindows;
	gboolean AllowModalDialogs;

	gboolean EnableWriteConsoleMessagesToStdout;

	gboolean EnableWebGL;

	gboolean Decorated;
	gboolean Resizable;

	int Width;
	int Height;

} settings_t;

static GtkWidget *create_window(gboolean offscreen) {
	if (gtk_init_check(0, NULL) == FALSE) return NULL;
	if(offscreen) return gtk_offscreen_window_new();
	return gtk_window_new(GTK_WINDOW_TOPLEVEL);
}

static WebKitWebView *init_window(GtkWidget *window, const char *title, const char *user_agent, settings_t *s, guint64 parent) {
	gtk_widget_hide_on_delete(window);
	gtk_window_set_title(GTK_WINDOW(window), title);
	gtk_window_set_decorated(GTK_WINDOW(window), s->Decorated);


	WebKitSettings *settings = webkit_settings_new();
	webkit_settings_set_enable_java(settings, s->EnableJava);
	webkit_settings_set_enable_plugins(settings, FALSE);
	webkit_settings_set_enable_frame_flattening(settings, s->EnableFrameFlattening);
	webkit_settings_set_enable_smooth_scrolling(settings, s->EnableSmoothScrolling);

	webkit_settings_set_enable_javascript(settings, s->EnableJavaScript);
	webkit_settings_set_javascript_can_open_windows_automatically(settings, s->EnableJavaScriptCanOpenWindows);
	webkit_settings_set_allow_modal_dialogs(settings, s->AllowModalDialogs);

	webkit_settings_set_enable_write_console_messages_to_stdout(settings, s->EnableWriteConsoleMessagesToStdout);
	webkit_settings_set_enable_webgl (settings, s->EnableWebGL);

	if(user_agent != NULL) webkit_settings_set_user_agent(settings, user_agent);

	if (s->Resizable) gtk_window_set_default_size(GTK_WINDOW(window), s->Width, s->Height);
	gtk_window_set_resizable(GTK_WINDOW(window), s->Resizable);
	gtk_widget_set_size_request(window,  s->Width, s->Height);
	gtk_window_set_position(GTK_WINDOW(window), GTK_WIN_POS_CENTER);

	WebKitWebView *webview = (WebKitWebView *)webkit_web_view_new_with_settings(settings);

	g_signal_connect(window, "delete-event", G_CALLBACK(window_close_cb), (void*)parent);
	g_signal_connect(webview, "load-changed", G_CALLBACK(wv_load_changed_cb),  (void*)parent);
	g_signal_connect(webview, "context-menu", G_CALLBACK(wv_context_menu_cb),  (void*)parent);


	GtkWidget *scroller = gtk_scrolled_window_new(NULL, NULL);
	gtk_container_add(GTK_CONTAINER(window), scroller);
	gtk_container_add(GTK_CONTAINER(scroller), GTK_WIDGET(webview));
	startHandler(parent);
	gtk_widget_show_all(window);

	return webview;
}

static void javascript_finished (GObject *object, GAsyncResult *result, gpointer data){
	GError *error = NULL;
	WebKitJavascriptResult *js_result = webkit_web_view_run_javascript_finish (WEBKIT_WEB_VIEW (object), result, &error);
	if (!js_result) {
		jsCallback((guint64)data, NULL, NULL, error->message);
		g_error_free (error);
		return;
	}

	JSGlobalContextRef context = webkit_javascript_result_get_global_context (js_result);
	JSValueRef value = webkit_javascript_result_get_value (js_result);
	jsCallback((guint64)data, context, value, NULL);
	webkit_javascript_result_unref (js_result);
}

static void execute_javascript(WebKitWebView *wv, guint64 cbID, const char *js) {
	webkit_web_view_run_javascript (wv, js, NULL, javascript_finished, (gpointer) cbID);
}

static const char * js_get_str(JSGlobalContextRef ctx, JSValueRef ref) {
	JSStringRef js_str_value;
	gchar      *str_value;
	gsize       str_length;

	js_str_value = JSValueToStringCopy (ctx, ref, NULL);
	str_length = JSStringGetMaximumUTF8CStringSize (js_str_value);
	str_value = (gchar *)g_malloc (str_length);
	JSStringGetUTF8CString (js_str_value, str_value, str_length);
	JSStringRelease (js_str_value);
	// g_print ("Script result: %s\n", str_value);
	// g_free (str_value);
	return str_value;
}

static void close_window(WebKitWebView *wv, GtkWidget *win) {
	webkit_web_view_try_close(wv);
	gtk_widget_destroy(win);
}

static void load_uri(WebKitWebView *wv, const char *uri) {
	webkit_web_view_load_uri(wv, uri);
}

static void load_html(WebKitWebView *wv, const char *html) {
	webkit_web_view_load_html(wv, html, "");
}

static void set_prop(WebKitSettings *s, const char * prop, gboolean v) {
	g_object_set (G_OBJECT(s), prop, v, NULL);
}
#endif /* WEBVIEW_H */
