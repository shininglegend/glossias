import { log } from 'console'
import styles from './page.module.css'

// Mark component as async since we're fetching data
export default async function Home() {
  // Fetch stories from our API route
  const res = await fetch('http://localhost:3000/api/stories', { 
    // Ensure fresh data on each page load
    cache: 'no-store'
  })
  const { Stories } = await res.json()
  
  log('Stories fetched:', Stories)
  log('Page loaded successfully')

  return (
    <>
      <header>
        <h1>Logos Stories</h1>
        <p>Select a story to begin reading</p>
        <hr />
      </header>
      
      <main className="container">
        <div className="stories-list">
          {Stories?.map((story: {
            ID: number,
            Title: string,
            WeekNumber: number,
            DayLetter: string
          }) => (
            <div key={story.ID} className="story-item">
              <h2>{story.Title}</h2>
              <p>Week {story.WeekNumber}{story.DayLetter}</p>
              <a href={`/stories/${story.ID}/page1`}>Start Reading</a>
            </div>
          ))}
        </div>
      </main>
    </>
  )
}