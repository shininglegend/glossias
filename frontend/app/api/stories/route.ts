// frontend/app/api/stories/route.ts
export async function GET() {
    const res = await fetch('http://localhost:8080/api/stories')
    const data = await res.json()
    return Response.json(data)
  }