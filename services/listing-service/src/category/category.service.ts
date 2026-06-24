import { ConflictException, Injectable, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { Category } from './category.entity';

@Injectable()
export class CategoryService {
  constructor(
    @InjectRepository(Category)
    private readonly repo: Repository<Category>,
  ) {}

  findAll(): Promise<Category[]> {
    return this.repo.find({ order: { name: 'ASC' } });
  }

  async findById(id: string): Promise<Category> {
    const category = await this.repo.findOne({ where: { id } });
    if (!category) {
      throw new NotFoundException('category not found');
    }
    return category;
  }

  async create(name: string, slug: string, parentId: string | null): Promise<Category> {
    const exists = await this.repo.findOne({ where: { slug } });
    if (exists) {
      throw new ConflictException('category slug already exists');
    }
    const category = this.repo.create({ name, slug, parentId });
    return this.repo.save(category);
  }
}
