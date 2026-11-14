import { afterEach, describe, expect, it, vi } from 'vitest'
import request from 'supertest'
import { app, userModel } from './app.js'

afterEach(() => {
     vi.restoreAllMocks()
})

describe('GET /api/user/all', () => {
     it('returns confirmed users', async () => {
          const users = [{ _id: 'user-1' }]
          const sort = vi.fn().mockResolvedValue(users)
          const limit = vi.fn().mockReturnValue({ sort })
          const find = vi.spyOn(userModel, 'find').mockReturnValue({ limit })

          const response = await request(app).get('/api/user/all')

          expect(response.status).toBe(200)
          expect(response.body.status).toBe('OK')
          expect(response.body.data).toEqual(users)
          expect(find).toHaveBeenCalledWith({ status: 'Confirmed' }, { _id: 1 })
          expect(limit).toHaveBeenCalledWith(10)
          expect(sort).toHaveBeenCalledWith({ createdAt: -1 })
     })

     it('returns an error when the user query fails', async () => {
          vi.spyOn(userModel, 'find').mockImplementation(() => {
               throw new Error('Database unavailable')
          })
          vi.spyOn(console, 'error').mockImplementation(() => { })

          const response = await request(app).get('/api/user/all')

          expect(response.status).toBe(500)
          expect(response.body).toEqual({ code: 500, status: 'Error' })
     })
})
