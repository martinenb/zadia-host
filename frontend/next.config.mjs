/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${process.env.BACKEND_URL || "http://host.docker.internal:8080"}/api/:path*`,
      },
    ]
  },
}
export default nextConfig
