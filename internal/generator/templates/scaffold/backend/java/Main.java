import com.sun.net.httpserver.HttpServer;
import com.sun.net.httpserver.HttpExchange;
import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;

public class Main {
    public static void main(String[] args) throws IOException {
        String portStr = System.getenv("BACKEND_PORT");
        int port = (portStr != null && !portStr.isEmpty()) ? Integer.parseInt(portStr) : 8080;

        HttpServer server = HttpServer.create(new InetSocketAddress("0.0.0.0", port), 0);
        server.createContext("/", exchange -> sendJson(exchange, "{\"message\": \"Hello from dev-box API (Java)\"}"));
        server.createContext("/health", exchange -> sendJson(exchange, "{\"status\": \"ok\"}"));
        server.start();
        System.out.println("Server running on port " + port);
    }

    private static void sendJson(HttpExchange exchange, String json) throws IOException {
        exchange.getResponseHeaders().set("Content-Type", "application/json");
        byte[] bytes = json.getBytes();
        exchange.sendResponseHeaders(200, bytes.length);
        try (OutputStream os = exchange.getResponseBody()) { os.write(bytes); }
    }
}
